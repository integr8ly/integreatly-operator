// postgres metric provider scrapes metrics for a single postgres (rds) instance
//
// we are required to gather data from postgres (rds) instances which can be used in valuable
// alerts, to ensure and monitor performance of postgres (rds) instances
//
// this providers does
//   - scrape metric data from cloudwatch
//   - build a generic cloud metric data type from cloudwatch data
//   - return generic cloud metric data to metric controller to be exposed
//
// this provider does not
//   - expose the metrics, this is controller at a higher level (controller)
package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudWatchTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	postgresMetricProviderName = "aws postgres metrics provider"
	cloudWatchRDSDBDimension   = "DBInstanceIdentifier"
)

var _ providers.PostgresMetricsProvider = (*PostgresMetricsProvider)(nil)

type PostgresMetricsProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewAWSPostgresMetricsProvider(client client.Client, logger *logrus.Entry) (*PostgresMetricsProvider, error) {
	cm, err := NewCredentialManager(client)
	if err != nil {
		return nil, err
	}
	return &PostgresMetricsProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"providers": postgresMetricProviderName}),
		CredentialManager: cm,
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}, nil
}

func (p *PostgresMetricsProvider) SupportsStrategy(strategy string) bool {
	return strategy == providers.AWSDeploymentStrategy
}

// ScrapeMetrics returns scraped metrics to metric controller
func (p PostgresMetricsProvider) ScrapePostgresMetrics(ctx context.Context, postgres *v1alpha1.Postgres, metricTypes []providers.CloudProviderMetricType) (*providers.ScrapeMetricsData, error) {
	logger := resources.NewActionLoggerWithFields(p.Logger, map[string]interface{}{
		resources.LoggingKeyAction: "ScrapeMetrics",
		"Resource":                 postgres.Name,
	})
	logger.Infof("reconciling postgres metrics %s", postgres.Name)

	// read storage strategy for postgres instance
	// this is required to create the correct credentials for aws
	postgresStrategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, postgres.Spec.Tier)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to read postgres aws strategy config")
	}

	// reconcile aws credentials (keys)
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, postgres.Namespace)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to reconcile rds credentials")
	}

	// create a session from postgres strategy (region) and reconciled aws keys
	cfg, err := CreateConfigFromStrategy(ctx, p.Client, providerCreds, postgresStrategyConfig)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to create aws session to scrape rds cloud watch metrics")
	}

	cloudwatchClient := NewCloudWatchClient(*cfg)
	// scrape metric data from cloud watch
	cloudMetrics, err := p.scrapeRDSCloudWatchMetricData(ctx, cloudwatchClient, postgres, metricTypes)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to scrape rds cloud watch metrics")
	}

	return &providers.ScrapeMetricsData{
		Metrics: cloudMetrics,
	}, nil
}

// scrapeRDSCloudWatchMetricData fetches cloud watch metrics for rds
// and parses it to a GenericCloudMetric in order to return to the controller
func (p *PostgresMetricsProvider) scrapeRDSCloudWatchMetricData(ctx context.Context, cloudWatchClient CloudWatchAPI, postgres *v1alpha1.Postgres, metricTypes []providers.CloudProviderMetricType) ([]*providers.GenericCloudMetric, error) {
	resourceID, err := resources.BuildInfraNameFromObject(ctx, p.Client, postgres.ObjectMeta, defaultAwsIdentifierLength)
	if err != nil {
		return nil, errorUtil.Errorf("error occurred building instance name: %v", err)
	}

	// getMetricData, returns multiple metrics and corresponding statistics in a singular api call
	// for more info see https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetMetricData.html
	logger := resources.NewActionLogger(p.Logger, "scrapeRDSCloudWatchMetricData")
	logger.Infof("scraping rds instance %s cloud watch metrics", resourceID)
	metricOutput, err := cloudWatchClient.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		// build metric data query array from `metricTypes`
		MetricDataQueries: buildRDSMetricDataQuery(metricTypes, resourceID),
		// metrics gathered from start time to end time
		StartTime: aws.Time(time.Now().Add(-resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration))),
		EndTime:   aws.Time(time.Now()),
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting metric for rds")
	}

	// get cluster if for use in metric labels
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting clusterID")
	}

	// ensure metric data results are not nil
	if len(metricOutput.MetricDataResults) == 0 {
		return nil, errorUtil.New("no metric data returned from rds cloudwatch")
	}

	logger.Infof("parsing rds cloud watch metrics for postgres %s", resourceID)
	// parse the returned data from the cloudwatch to a GenericCloudMetric
	var metrics []*providers.GenericCloudMetric
	for _, metricData := range metricOutput.MetricDataResults {
		// status code complete ensures all metrics have been successful
		if metricData.StatusCode != cloudWatchTypes.StatusCodeComplete {
			continue
		}
		// depending on the number of data points, several values can be returned
		for _, value := range metricData.Values {
			// convert aws metric data to generic cloud metric data
			metrics = append(metrics, &providers.GenericCloudMetric{
				Name: *metricData.Id,
				Labels: map[string]string{
					resources.LabelClusterIDKey:   clusterID,
					resources.LabelResourceIDKey:  postgres.Name,
					resources.LabelNamespaceKey:   postgres.Namespace,
					resources.LabelInstanceIDKey:  resourceID,
					resources.LabelProductNameKey: postgres.Labels["productName"],
					resources.LabelStrategyKey:    postgresProviderName,
				},
				Value: value,
			})
		}
	}
	return metrics, nil
}

// buildRDSMetricDataQuery builds an aws query from wanted rds metric types
func buildRDSMetricDataQuery(metricTypes []providers.CloudProviderMetricType, resourceID string) []cloudWatchTypes.MetricDataQuery {
	var metricDataQueries []cloudWatchTypes.MetricDataQuery
	for _, metricType := range metricTypes {
		metricDataQueries = append(metricDataQueries, cloudWatchTypes.MetricDataQuery{
			// id needs to be unique, and is built from the metric name and type
			// the metric name is converted from camel case to snake case to allow it to be easily reused when exposing the metric
			Id: aws.String(metricType.PrometheusMetricName),
			MetricStat: &cloudWatchTypes.MetricStat{
				Metric: &cloudWatchTypes.Metric{
					MetricName: aws.String(metricType.ProviderMetricName),
					Namespace:  aws.String("AWS/RDS"),
					Dimensions: []cloudWatchTypes.Dimension{
						{
							Name:  aws.String(cloudWatchRDSDBDimension),
							Value: aws.String(resourceID),
						},
					},
				},
				Stat:   aws.String(metricType.Statistic),
				Period: aws.Int32(int32(resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration).Seconds())),
			},
		})
	}
	return metricDataQueries
}
