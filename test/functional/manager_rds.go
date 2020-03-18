package functional

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/integreatly-operator/test/common"
	"github.com/sirupsen/logrus"
)

var _ ClusterResourceManager = &RDSInstanceManager{}

const (
	loggingKeyDatabase = "database-id"
)

type RDSInstanceManager struct {
	rdsClient rdsClient
	logger    *logrus.Entry
}

func (r *RDSInstanceManager) GetName() string {
	return "AWS RDS Manager"
}
func NewDefaultRDSInstanceManager(session *session.Session, logger *logrus.Entry) *RDSInstanceManager {
	return &RDSInstanceManager{
		rdsClient: rds.New(session),
		logger:    logger.WithField("engine", managerRDS),
	}
}

func (r *RDSInstanceManager) GetResourcesForCluster(clusterID string, tags map[string]string) ([]*ResourceOutput, error) {
	r.logger.Debug("Looking for matching resources")
	clusterDescribeInput := &rds.DescribeDBInstancesInput{}
	clusterDescribeOutput, err := r.rdsClient.DescribeDBInstances(clusterDescribeInput)
	if err != nil {
		return nil, common.WrapLog(err, "error getting clusterOutput", r.logger)
	}
	var dbInstanceToReturn []*ResourceOutput
	for _, dbInstance := range clusterDescribeOutput.DBInstances {
		dbLogger := r.logger.WithField(loggingKeyDatabase, aws.StringValue(dbInstance.DBInstanceIdentifier))
		dbLogger.Debugf("checking tags database cluster")
		tagListInput := &rds.ListTagsForResourceInput{
			ResourceName: dbInstance.DBInstanceArn,
		}
		tagListOutput, err := r.rdsClient.ListTagsForResource(tagListInput)
		if err != nil {
			return nil, common.WrapLog(err, "failed to list tags for database cluster", r.logger)
		}
		dbLogger.Debugf("checking for cluster tag match (%s=%s) on database", tagKeyClusterID, clusterID)
		if common.FindTag(tagKeyClusterID, clusterID, tagListOutput.TagList) == nil {
			dbLogger.Debugf("database did not contain cluster tag match (%s=%s)", tagKeyClusterID, clusterID)
			continue
		}
		extraTagsMatch := true
		for extraTagKey, extraTagVal := range tags {
			dbLogger.Debugf("checking for additional tag match (%s=%s) on database", extraTagKey, extraTagVal)
			if common.FindTag(extraTagKey, extraTagVal, tagListOutput.TagList) == nil {
				extraTagsMatch = false
				break
			}
		}
		if !extraTagsMatch {
			dbLogger.Debug("additional tags did not match, ignoring database")
			continue
		}
		matchingInstance := &ResourceOutput{
			Name:        aws.StringValue(dbInstance.DBName),
			ResourceARN: aws.StringValue(dbInstance.DBInstanceArn),
		}
		dbInstanceToReturn = append(dbInstanceToReturn, matchingInstance)

	}
	if dbInstanceToReturn != nil {
		return dbInstanceToReturn, nil
	}
	return nil, nil
}
