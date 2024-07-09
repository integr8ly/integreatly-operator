package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/redis/apiv1/redispb"
	stratType "github.com/integr8ly/cloud-resource-operator/pkg/client/types"
	croGCP "github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	errorUtil "github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/genproto/googleapis/type/timeofday"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utils "k8s.io/utils/ptr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const defaultGcpStratValue = `{"development": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }}`

type StrategyProvider struct {
	Client    k8sclient.Client
	Tier      string
	Namespace string
}

func NewGCPStrategyProvider(client k8sclient.Client, tier string, namespace string) *StrategyProvider {
	return &StrategyProvider{
		Client:    client,
		Tier:      tier,
		Namespace: namespace,
	}
}

func (p *StrategyProvider) ReconcileStrategyMap(ctx context.Context, client k8sclient.Client, timeConfig *stratType.StrategyTimeConfig) error {
	gcpStratConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      croGCP.DefaultConfigMapName,
			Namespace: p.Namespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, client, gcpStratConfig, func() error {
		// ensure data is not nil, create a default config map
		if gcpStratConfig.Data == nil {
			// default config map contains a `development` and `production` tier
			gcpStratConfig.Data = croGCP.BuildDefaultConfigMap(gcpStratConfig.Name, gcpStratConfig.Namespace).Data
		}

		// check to ensure postgres and redis key is included in config data
		// build default postgres and redis key, contains a `development` and `production` tier
		if _, ok := gcpStratConfig.Data[stratType.PostgresStratKey]; !ok {
			gcpStratConfig.Data[stratType.PostgresStratKey] = defaultGcpStratValue
		}
		if _, ok := gcpStratConfig.Data[stratType.RedisStratKey]; !ok {
			gcpStratConfig.Data[stratType.RedisStratKey] = defaultGcpStratValue
		}

		// marshal strategies, updating existing strategies with new values
		postgresStrategy, err := p.reconcilePostgresStrategy(gcpStratConfig.Data[stratType.PostgresStratKey], timeConfig)
		if err != nil {
			return errorUtil.Wrapf(err, "failed to reconcile postgres strategy")
		}
		redisStrategy, err := p.reconcileRedisStrategy(gcpStratConfig.Data[stratType.RedisStratKey], timeConfig)
		if err != nil {
			return errorUtil.Wrapf(err, "failed to reconcile redis strategy")
		}

		// setting postgres and redis values to be updated strategies
		gcpStratConfig.Data[stratType.PostgresStratKey] = postgresStrategy
		gcpStratConfig.Data[stratType.RedisStratKey] = redisStrategy

		return nil
	}); err != nil {
		return fmt.Errorf("failed to update gcp strategy config map : %v", err)
	}
	return nil
}

func (p *StrategyProvider) reconcilePostgresStrategy(config string, timeConfig *stratType.StrategyTimeConfig) (string, error) {
	var rawStrategy map[string]*croGCP.StrategyConfig
	if err := json.Unmarshal([]byte(config), &rawStrategy); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for postgres resource")
	}
	postgresCreateConfig := &croGCP.CreateInstanceRequest{}
	if err := json.Unmarshal(rawStrategy[p.Tier].CreateStrategy, postgresCreateConfig); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal gcp postgres cluster config")
	}
	if postgresCreateConfig.Instance == nil {
		postgresCreateConfig.Instance = &gcpiface.DatabaseInstance{
			Settings: &gcpiface.Settings{},
		}
	}
	if postgresCreateConfig.Instance.Settings == nil {
		postgresCreateConfig.Instance.Settings = &gcpiface.Settings{}
	}

	day := int64(7)
	if timeConfig.MaintenanceStartTime.Day != time.Sunday {
		day = int64(timeConfig.MaintenanceStartTime.Day)
	}
	hour := int64(timeConfig.MaintenanceStartTime.Time.Hour())
	if postgresCreateConfig.Instance.Settings.MaintenanceWindow == nil ||
		utils.Deref(postgresCreateConfig.Instance.Settings.MaintenanceWindow.Day, -1) != day ||
		utils.Deref(postgresCreateConfig.Instance.Settings.MaintenanceWindow.Hour, -1) != hour {
		postgresCreateConfig.Instance.Settings.MaintenanceWindow = &gcpiface.MaintenanceWindow{
			Day:  utils.To(day),
			Hour: utils.To(hour),
		}
	}

	startTime := fmt.Sprintf("%02d:%02d", timeConfig.BackupStartTime.Time.Hour(), timeConfig.BackupStartTime.Time.Minute())
	if postgresCreateConfig.Instance.Settings.BackupConfiguration == nil ||
		postgresCreateConfig.Instance.Settings.BackupConfiguration.StartTime != startTime {
		postgresCreateConfig.Instance.Settings.BackupConfiguration = &gcpiface.BackupConfiguration{
			StartTime: startTime,
		}
	}

	// marshall the create db instance input struct to json
	postgresMarshalledConfig, err := json.Marshal(postgresCreateConfig)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal gcp postgres cluster config")
	}

	// set the createstrategy on our expected tier to the marshalled create db instance input struct
	rawStrategy[p.Tier].CreateStrategy = postgresMarshalledConfig

	// marshall the entire strategy back to json
	marshalledStrategy, err := json.Marshal(rawStrategy)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal gcp strategy")
	}

	return string(marshalledStrategy), nil
}

func (p *StrategyProvider) reconcileRedisStrategy(config string, timeConfig *stratType.StrategyTimeConfig) (string, error) {
	var rawStrategy map[string]*croGCP.StrategyConfig
	if err := json.Unmarshal([]byte(config), &rawStrategy); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for redis resource")
	}

	redisCreateConfig := &redispb.CreateInstanceRequest{}
	if err := json.Unmarshal(rawStrategy[p.Tier].CreateStrategy, redisCreateConfig); err != nil {
		return "", errorUtil.Wrapf(err, "failed to unmarshal gcp redis cluster config")
	}

	if redisCreateConfig.Instance == nil {
		redisCreateConfig.Instance = &redispb.Instance{}
	}
	if redisCreateConfig.Instance.MaintenancePolicy == nil ||
		redisCreateConfig.Instance.MaintenancePolicy.WeeklyMaintenanceWindow == nil {
		redisCreateConfig.Instance.MaintenancePolicy = &redispb.MaintenancePolicy{
			WeeklyMaintenanceWindow: make([]*redispb.WeeklyMaintenanceWindow, 1),
		}
	}
	maintenanceDay := dayofweek.DayOfWeek(7)
	if timeConfig.MaintenanceStartTime.Day != time.Sunday {
		maintenanceDay = dayofweek.DayOfWeek(timeConfig.MaintenanceStartTime.Day)
	}

	maintenanceTime := &timeofday.TimeOfDay{
		Hours:   int32(timeConfig.MaintenanceStartTime.Time.Hour()),
		Minutes: int32(timeConfig.MaintenanceStartTime.Time.Minute()),
	}
	if redisCreateConfig.Instance.MaintenancePolicy.WeeklyMaintenanceWindow[0] == nil ||
		redisCreateConfig.Instance.MaintenancePolicy.WeeklyMaintenanceWindow[0].Day != maintenanceDay ||
		redisCreateConfig.Instance.MaintenancePolicy.WeeklyMaintenanceWindow[0].StartTime != maintenanceTime {
		redisCreateConfig.Instance.MaintenancePolicy.WeeklyMaintenanceWindow[0] = &redispb.WeeklyMaintenanceWindow{
			Day:       maintenanceDay,
			StartTime: maintenanceTime,
		}
	}

	// marshall the create db instance input struct to json
	redisMarshalledConfig, err := json.Marshal(redisCreateConfig)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal gcp redis cluster config")
	}

	// set the createstrategy on our expected tier to the marshalled create db instance input struct
	rawStrategy[p.Tier].CreateStrategy = redisMarshalledConfig

	// marshall the entire strategy back to json
	marshalledStrategy, err := json.Marshal(rawStrategy)
	if err != nil {
		return "", errorUtil.Wrapf(err, "failed to marshal gcp strategy")
	}

	// return json in string format
	return string(marshalledStrategy), nil
}
