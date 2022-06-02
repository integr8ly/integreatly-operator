/*
AWS Strategy Setup

A utility to abstract the various strategy map ConfigMaps from the service using CRO

Problem Statement:
 - We require an AWS strategy map ConfigMaps to be in place to provide configuration used to provision AWS cloud resources

This utility provides the abstraction necessary, provisioning an AWS strategy map

We accept start times for maintenance and backup times as a level of abstraction
Building the correct maintenance and backup times necessary for AWS
Maintenance Window format ddd:hh:mm-ddd:hh:mm
Backup/Snapshot Window hh:mm-hh:mm
*/
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	croAWS "github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"
)

// reconciles aws strategy map, adding maintenance and backup window fields
func reconcileAWSStrategyMap(ctx context.Context, client client.Client, timeConfig *StrategyTimeConfig, tier, namespace string) error {
	// build backup and maintenance windows
	backupWindow, maintenanceWindow, err := buildAWSWindows(timeConfig)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to build aws windows")
	}

	// create or update aws strategies config map
	awsStratConfig := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      croAWS.DefaultConfigMapName,
			Namespace: namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, client, awsStratConfig, func() error {
		// ensure data is not nil, create a default config map
		if awsStratConfig.Data == nil {
			// default config map contains a `development` and `production` tier
			awsStratConfig.Data = croAWS.BuildDefaultConfigMap(awsStratConfig.Name, awsStratConfig.Namespace).Data
		}

		// check to ensure postgres and redis key is included in config data
		// build default postgres and redis key, contains a `development` and `production` tier
		if _, ok := awsStratConfig.Data[postgresStratKey]; !ok {
			awsStratConfig.Data[postgresStratKey] = buildDefaultAWSStratValue()
		}
		if _, ok := awsStratConfig.Data[redisStratKey]; !ok {
			awsStratConfig.Data[redisStratKey] = buildDefaultAWSStratValue()
		}

		// marshal strategies, updating existing strategies with new values
		postgresStrategy, err := reconcilePostgresStrategy(awsStratConfig.Data[postgresStratKey], tier, backupWindow, maintenanceWindow)
		if err != nil {
			return errorUtil.Wrapf(err, "failed to reconcile postgres strategy")
		}
		redisStrategy, err := reconcileRedisStrategy(awsStratConfig.Data[redisStratKey], tier, backupWindow, maintenanceWindow)
		if err != nil {
			return errorUtil.Wrapf(err, "failed to reconcile redis strategy")
		}

		// setting postgres and redis values to be updated strategies
		awsStratConfig.Data[postgresStratKey] = postgresStrategy
		awsStratConfig.Data[redisStratKey] = redisStrategy

		return nil
	}); err != nil {
		return fmt.Errorf("failed to update aws strategy config map : %v", err)
	}
	return nil
}

// reconciles Postgres strategy
// unmarshalls raw strategy to create db instance input
// checks found values from current config map vs expected values
// marshalls create db instance input back to raw strategy
func reconcilePostgresStrategy(config, tier, backupWindow, maintenanceWindow string) (string, error) {
	// unmarshall config data to type of cro aws strategy config
	var rawStrategy map[string]*croAWS.StrategyConfig
	if err := json.Unmarshal([]byte(config), &rawStrategy); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for postgres resource")
	}

	// unmarshall the create strategy from the raw strategy tier to type create db instance input
	rdsCreateConfig := &rds.CreateDBInstanceInput{}
	if err := json.Unmarshal(rawStrategy[tier].CreateStrategy, rdsCreateConfig); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal aws rds cluster config")
	}

	// check create db instance input against expected values
	// backup window and maintenance window are expected values via RHMIconfig
	// todo we may want to handle more strategy input, we should add functionality to check every value
	if rdsCreateConfig.PreferredBackupWindow == nil || *rdsCreateConfig.PreferredBackupWindow != backupWindow {
		rdsCreateConfig.PreferredBackupWindow = aws.String(backupWindow)
	}
	if rdsCreateConfig.PreferredMaintenanceWindow == nil || *rdsCreateConfig.PreferredMaintenanceWindow != maintenanceWindow {
		rdsCreateConfig.PreferredMaintenanceWindow = aws.String(maintenanceWindow)
	}

	// marshall the create db instance input struct to json
	rdsMarshalledConfig, err := json.Marshal(rdsCreateConfig)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal aws rds cluster config")
	}

	// set the createstrategy on our expected tier to the marshalled create db instance input struct
	rawStrategy[tier].CreateStrategy = rdsMarshalledConfig

	// marshall the entire strategy back to json
	marshalledStrategy, err := json.Marshal(rawStrategy)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal aws strategy")
	}

	// return json in string format
	return string(marshalledStrategy), nil
}

// reconciles Redis strategy
// unmarshalls raw strategy to create db instance input
// checks found values from current config map vs expected values
// marshalls create db instance input back to raw strategy
func reconcileRedisStrategy(config, tier, backupTimeStart, maintenanceTimeStart string) (string, error) {
	// unmarshall config data to type of cro aws strategy config
	var rawStrategy map[string]*croAWS.StrategyConfig
	if err := json.Unmarshal([]byte(config), &rawStrategy); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for postgres resource")
	}

	// unmarshall the create strategy from the raw strategy tier to type create db instance input
	elasticacheCreateConfig := &elasticache.CreateReplicationGroupInput{}
	if err := json.Unmarshal(rawStrategy[tier].CreateStrategy, elasticacheCreateConfig); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal aws rds cluster config")
	}

	// check create db instance input against expected values
	// snapshot window and maintenance window are expected values via config
	// todo we may want to handle more strategy input, we should add functionality to check every value
	if elasticacheCreateConfig.SnapshotWindow == nil || *elasticacheCreateConfig.SnapshotWindow != backupTimeStart {
		elasticacheCreateConfig.SnapshotWindow = aws.String(backupTimeStart)
	}
	if elasticacheCreateConfig.PreferredMaintenanceWindow == nil || *elasticacheCreateConfig.PreferredMaintenanceWindow != maintenanceTimeStart {
		elasticacheCreateConfig.PreferredMaintenanceWindow = aws.String(maintenanceTimeStart)
	}

	// marshall the create db instance input struct to json
	rdsMarshalledConfig, err := json.Marshal(elasticacheCreateConfig)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal aws rds cluster config")
	}

	// set the createstrategy on our expected tier to the marshalled create db instance input struct
	rawStrategy[tier].CreateStrategy = rdsMarshalledConfig

	// marshall the entire strategy back to json
	marshalledStrategy, err := json.Marshal(rawStrategy)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal aws strategy")
	}

	// return json in string format
	return string(marshalledStrategy), nil
}

// builder function to provide default data for aws strategy config map
func buildDefaultAWSConfigMap() map[string]string {
	return map[string]string{
		blobstorageStratKey: buildDefaultAWSStratValue(),
		redisStratKey:       buildDefaultAWSStratValue(),
		postgresStratKey:    buildDefaultAWSStratValue(),
	}
}

// builder function to provide default value used per tier in aws strategy config map
func buildDefaultAWSStratValue() string {
	return "{\"development\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }}"
}

// build aws maintenance and backup windows
func buildAWSWindows(timeConfig *StrategyTimeConfig) (string, string, error) {
	// ensure backup applyOn format is correct
	parsedBackupTime, err := time.Parse("15:04", timeConfig.BackupStartTime)
	if err != nil {
		return "", "", errorUtil.Wrapf(err, "failed to parse backup ApplyOn value")
	}

	// correct format is assumed by the time this function is called
	maintenanceWindowSegments := strings.Split(timeConfig.MaintenanceStartTime, " ")
	parsedMaintenanceTime, err := time.Parse("15:04", maintenanceWindowSegments[1])
	if err != nil {
		return "", "", errorUtil.Wrapf(err, "failed to parse maintenance ApplyFrom value")
	}

	// ensure ddd is correct and overlaps to the following day if block starts after 23 hours
	// set window day from value from segmented string
	maintenanceWindowDayFrom := strings.ToLower(maintenanceWindowSegments[0][:3])
	// map of valid day with time.weekday int value
	var validMaintenanceDays = map[string]int{
		"sun": 0,
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
		"fri": 5,
		"sat": 6,
	}

	// get current maintenance dayTo value from maintenance dayFrom value
	var maintenanceTimeTo int
	for day, value := range validMaintenanceDays {
		if maintenanceWindowDayFrom == day {
			maintenanceTimeTo = value
		}
	}

	// if the current maintenance time value over laps into the following day
	// set current maintenance dayTo value to the following day
	if parsedMaintenanceTime.Hour() == 23 {
		maintenanceTimeTo = 0
		if maintenanceWindowDayFrom != "sat" {
			maintenanceTimeTo = validMaintenanceDays[maintenanceWindowDayFrom] + 1
		}
	}

	// convert maintenance dayTo value to time.Weekday format to convert int back to string
	maintenanceWindowDayTo := time.Weekday(maintenanceTimeTo).String()
	// convert time.Weekday string to first 3 chars and to lower case, as is expected for AWS format
	maintenanceWindowDayTo = strings.ToLower(maintenanceWindowDayTo[:3])

	// set maintenance time plus one hour
	parsedMaintenanceTimePlusOneHour := parsedMaintenanceTime.Add(time.Hour)

	// add one hour to applyOn format
	parsedBackupTimePlusOneHour := parsedBackupTime.Add(time.Hour)

	// build expected aws maintenance format ddd:hh:mm-ddd:hh:mm
	awsMaintenanceString := fmt.Sprintf("%s:%02d:%02d-%s:%02d:%02d", maintenanceWindowDayFrom, parsedMaintenanceTime.Hour(), parsedMaintenanceTime.Minute(), maintenanceWindowDayTo, parsedMaintenanceTimePlusOneHour.Hour(), parsedMaintenanceTimePlusOneHour.Minute())
	// build expected aws backup format hh:mm-hh:mm
	awsBackupString := fmt.Sprintf("%02d:%02d-%02d:%02d", parsedBackupTime.Hour(), parsedBackupTime.Minute(), parsedBackupTimePlusOneHour.Hour(), parsedBackupTimePlusOneHour.Minute())

	// ensure backup and maintenance time ranges do not overlap
	// we expect RHOAM operator to validate the ranges, as a sanity check we preform an extra validation here
	// this is to avoid an obscure error message from AWS when we apply the times
	// http://baodad.blogspot.com/2014/06/date-range-overlap.html
	// (StartA <= EndB)  and  (EndA >= StartB)
	if timeBlockOverlaps(parsedBackupTime, parsedBackupTimePlusOneHour, parsedMaintenanceTime, parsedMaintenanceTimePlusOneHour) {
		return "", "", errors.New(fmt.Sprintf("backup and maintenance windows can not overlap, we require a minumum of 1 hour windows, current backup window : %s, current maintenance window : %s ", awsBackupString, awsMaintenanceString))
	}

	return awsBackupString, awsMaintenanceString, nil
}

func timeBlockOverlaps(startA, endA, startB, endB time.Time) bool {
	return startA.Unix() <= endB.Unix() && endA.Unix() >= startB.Unix()
}
