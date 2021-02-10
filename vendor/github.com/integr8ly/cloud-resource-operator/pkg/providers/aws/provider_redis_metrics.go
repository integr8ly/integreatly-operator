package aws

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	redisMetricProviderName        = "aws elasticache metrics provider"
	cloudWatchElastiCacheDimension = "CacheClusterId"
)

var _ providers.RedisMetricsProvider = (*RedisMetricsProvider)(nil)

type RedisMetricsProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewAWSRedisMetricsProvider(client client.Client, logger *logrus.Entry) *RedisMetricsProvider {
	return &RedisMetricsProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"providers": redisMetricProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}
}

func (r *RedisMetricsProvider) SupportsStrategy(strategy string) bool {
	return strategy == providers.AWSDeploymentStrategy
}

func (r *RedisMetricsProvider) ScrapeRedisMetrics(ctx context.Context, redis *v1alpha1.Redis, metricTypes []providers.CloudProviderMetricType) (*providers.ScrapeMetricsData, error) {
	logger := resources.NewActionLoggerWithFields(r.Logger, map[string]interface{}{
		resources.LoggingKeyAction: "ScrapeMetrics",
		"Resource":                 redis.Name,
	})
	logger.Infof("reconciling redis metrics %s", redis.Name)

	// read storage strategy for redis instance
	// this is required to create the correct credentials for aws
	redisStrategyConfig, err := r.ConfigManager.ReadStorageStrategy(ctx, providers.RedisResourceType, redis.Spec.Tier)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to read redis aws strategy config")
	}

	// reconcile aws credentials (keys)
	providerCreds, err := r.CredentialManager.ReconcileProviderCredentials(ctx, redis.Namespace)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to reconcile elasticache credentials")
	}

	// create a session from redis strategy (region) and reconciled aws keys
	sess, err := CreateSessionFromStrategy(ctx, r.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, redisStrategyConfig)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to create aws session to scrape elasticache cloud watch metrics")
	}

	// scrape metric data from cloud watch
	cloudMetrics, err := r.scrapeRedisCloudWatchMetricData(ctx, cloudwatch.New(sess), redis, elasticache.New(sess), metricTypes)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to scrape elasticache cloud watch metrics")
	}

	return &providers.ScrapeMetricsData{
		Metrics: cloudMetrics,
	}, nil
}

func (r *RedisMetricsProvider) scrapeRedisCloudWatchMetricData(ctx context.Context, cloudWatchApi cloudwatchiface.CloudWatchAPI, redis *v1alpha1.Redis, elastiCacheApi elasticacheiface.ElastiCacheAPI, metricTypes []providers.CloudProviderMetricType) ([]*providers.GenericCloudMetric, error) {
	resourceID, err := BuildInfraNameFromObject(ctx, r.Client, redis.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return nil, errorUtil.Errorf("error occurred building instance name: %v", err)
	}

	// getMetricData, returns multiple metrics and corresponding statistics in a singular api call
	// for more info see https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetMetricData.html
	logger := resources.NewActionLogger(r.Logger, "scrapeRedisCloudWatchMetricData")
	logger.Infof("scraping redis instance %s cloud watch metrics", resourceID)

	// get cluster if for use in metric labels
	clusterID, err := resources.GetClusterID(ctx, r.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting clusterID")
	}

	var metrics []*providers.GenericCloudMetric
	listOutput, err := elastiCacheApi.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed redis metrics to describe replicationGroups")
	}
	replicationGroups := listOutput.ReplicationGroups
	// Metrics are returned per node for ElastiCache
	var foundCache *elasticache.ReplicationGroup
	for _, c := range replicationGroups {
		if *c.ReplicationGroupId == resourceID {
			foundCache = c
			break
		}
	}
	if foundCache == nil {
		return nil, errorUtil.Errorf("redis metrics failed to find cache in replication group")
	}

	// poll MemberCluster array for CacheClusterId
	for _, cacheClusterId := range foundCache.MemberClusters {
		metricOutput, err := cloudWatchApi.GetMetricData(&cloudwatch.GetMetricDataInput{
			// build metric data query array from `metricType`
			MetricDataQueries: buildRedisMetricDataQuery(*cacheClusterId, metricTypes),
			// metrics gathered from start time to end time
			StartTime: aws.Time(time.Now().Add(-resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration))),
			EndTime:   aws.Time(time.Now()),
		})
		if err != nil {
			logger.Error(err, "error getting metric for elasticache")
			continue
		}

		// ensure metric data results are not nil
		if metricOutput.MetricDataResults == nil {
			logger.Error("no metric data returned from elasticache cloudwatch")
			continue
		}

		logger.Infof("parsing elasticache cloud watch metrics for redis %s", resourceID)
		// parse the returned data from the cloudwatch to a GenericCloudMetric

		for _, metricData := range metricOutput.MetricDataResults {
			// status code complete ensures all metrics have been successful
			if *metricData.StatusCode != cloudwatch.StatusCodeComplete {
				continue
			}
			// depending on the number of data points, several values can be returned
			for _, value := range metricData.Values {
				// convert aws metric data to generic cloud metric data
				metrics = append(metrics, &providers.GenericCloudMetric{
					Name: *metricData.Id,
					Labels: map[string]string{
						"clusterID":   clusterID,
						"resourceID":  redis.Name,
						"namespace":   redis.Namespace,
						"instanceID":  *cacheClusterId,
						"productName": redis.Labels["productName"],
						"strategy":    redisProviderName,
					},
					Value: *value,
				})
			}
		}

	}
	return metrics, nil

}

func buildRedisMetricDataQuery(cacheClusterId string, metricTypes []providers.CloudProviderMetricType) []*cloudwatch.MetricDataQuery {
	var metricDataQueries []*cloudwatch.MetricDataQuery
	for _, metricType := range metricTypes {
		metricDataQueries = append(metricDataQueries, &cloudwatch.MetricDataQuery{
			Id: aws.String(metricType.PromethuesMetricName),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					MetricName: aws.String(metricType.ProviderMetricName),
					Namespace:  aws.String("AWS/ElastiCache"),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String(cloudWatchElastiCacheDimension),
							Value: aws.String(cacheClusterId),
						},
					},
				},
				Stat:   aws.String(metricType.Statistic),
				Period: aws.Int64(int64(resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration).Seconds())),
			},
		})
	}
	return metricDataQueries
}
