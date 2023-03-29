package gcp

import (
	"context"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	postgresMetricProviderName   = "gcp-monitoring"
	postgresMetricFilterTemplate = "resource.type=%q resource.labels.database_id=%q metric.type=%q"
)

var _ providers.PostgresMetricsProvider = (*PostgresMetricsProvider)(nil)

type PostgresMetricsProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewGCPPostgresMetricsProvider(client client.Client, logger *logrus.Entry) (*PostgresMetricsProvider, error) {
	return &PostgresMetricsProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"providers": postgresMetricProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
	}, nil
}

func (p *PostgresMetricsProvider) SupportsStrategy(strategy string) bool {
	return strategy == providers.GCPDeploymentStrategy
}

func (p *PostgresMetricsProvider) ScrapePostgresMetrics(ctx context.Context, pg *v1alpha1.Postgres, metricTypes []providers.CloudProviderMetricType) (*providers.ScrapeMetricsData, error) {
	p.Logger.Infof("reconciling postgres metrics for instance %s", pg.Name)
	strategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, pg.Spec.Tier)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve postgres strategy config: %w", err)
	}
	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, pg.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile postgres provider credentials: %w", err)
	}
	clientOption := option.WithCredentialsJSON(creds.ServiceAccountJson)
	metricClient, err := gcpiface.NewMetricAPI(ctx, clientOption, p.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise metric client: %w", err)
	}
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster id: %w", err)
	}
	instanceID, err := resources.BuildInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, defaultGcpIdentifierLength)
	if err != nil {
		return nil, fmt.Errorf("error building instance id: %w", err)
	}
	allMetrics := make([]*providers.GenericCloudMetric, 0)
	opts := getMetricsOpts{
		filterTemplate:         postgresMetricFilterTemplate,
		monitoringResourceType: resources.MonitoringResourceTypeCloudsqlDatabase,
		projectID:              fmt.Sprintf("projects/%s", strategyConfig.ProjectID),
		instanceID:             fmt.Sprintf("%s:%s", strategyConfig.ProjectID, instanceID),
		defaultLabels:          resources.BuildGenericMetricLabels(pg.ObjectMeta, clusterID, instanceID, postgresProviderName),
	}
	for _, metric := range metricTypes {
		if resources.IsCompoundMetric(metric.PrometheusMetricName) {
			resultMetric, err := calculateAvailableMemory(ctx, metricClient, metric, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate postgres compound metric %s: %w", metric.PrometheusMetricName, err)
			}
			allMetrics = append(allMetrics, resultMetric)
			continue
		}
		opts.metricsToQuery = append(opts.metricsToQuery, metric)
	}
	remainingMetrics, err := getMetrics(ctx, metricClient, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres metrics data: %w", err)
	}
	allMetrics = append(allMetrics, remainingMetrics...)
	return &providers.ScrapeMetricsData{
		Metrics: allMetrics,
	}, nil
}
