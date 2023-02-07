package gcp

import (
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"context"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"sync"
	"time"
)

const (
	redisMetricProviderName   = "gcp-monitoring"
	redisMetricFilterTemplate = "resource.type=%q resource.labels.instance_id=%q metric.type=%q metric.label.role=primary"
)

var _ providers.RedisMetricsProvider = (*RedisMetricsProvider)(nil)

type RedisMetricsProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewGCPRedisMetricsProvider(client client.Client, logger *logrus.Entry) (*RedisMetricsProvider, error) {
	return &RedisMetricsProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"providers": redisMetricProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
	}, nil
}

func (p *RedisMetricsProvider) SupportsStrategy(strategy string) bool {
	return strategy == providers.GCPDeploymentStrategy
}

func (p *RedisMetricsProvider) ScrapeRedisMetrics(ctx context.Context, r *v1alpha1.Redis, metricTypes []providers.CloudProviderMetricType) (*providers.ScrapeMetricsData, error) {
	p.Logger.Infof("reconciling redis metrics for instance %s", r.Name)
	strategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.RedisResourceType, r.Spec.Tier)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve redis strategy config: %w", err)
	}
	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, r.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile redis provider credentials: %w", err)
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
	instanceID, err := resources.BuildInfraNameFromObject(ctx, p.Client, r.ObjectMeta, defaultGcpIdentifierLength)
	if err != nil {
		return nil, fmt.Errorf("error building instance id: %w", err)
	}
	allMetrics := make([]*providers.GenericCloudMetric, 0)
	opts := getMetricsOpts{
		filterTemplate:         redisMetricFilterTemplate,
		monitoringResourceType: resources.MonitoringResourceTypeRedisInstance,
		projectID:              fmt.Sprintf("projects/%s", strategyConfig.ProjectID),
		instanceID:             fmt.Sprintf(redisInstanceNameFormat, strategyConfig.ProjectID, strategyConfig.Region, instanceID),
		defaultLabels:          resources.BuildGenericMetricLabels(r.ObjectMeta, clusterID, instanceID, redisProviderName),
	}
	for _, metric := range metricTypes {
		if resources.IsCompoundMetric(metric.PrometheusMetricName) {
			resultMetric, err := calculateAvailableMemory(ctx, metricClient, metric, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate redis compound metric %s: %w", metric.PrometheusMetricName, err)
			}
			allMetrics = append(allMetrics, resultMetric)
			continue
		}
		if resources.IsComputedCpuMetric(metric.PrometheusMetricName) {
			resultMetric, err := calculateCpuUtilization(ctx, metricClient, metric, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate redis cpu utilization metric %s: %w", metric.ProviderMetricName, err)
			}
			allMetrics = append(allMetrics, resultMetric)
			continue
		}
		opts.metricsToQuery = append(opts.metricsToQuery, metric)
	}
	remainingMetrics, err := getMetrics(ctx, metricClient, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get redis metrics data: %w", err)
	}
	allMetrics = append(allMetrics, remainingMetrics...)
	return &providers.ScrapeMetricsData{
		Metrics: allMetrics,
	}, nil
}

type getMetricsOpts struct {
	metricsToQuery         []providers.CloudProviderMetricType
	filterTemplate         string
	monitoringResourceType resources.MonitoringResourceType
	projectID              string
	instanceID             string
	defaultLabels          map[string]string
}

func getMetrics(ctx context.Context, metricClient gcpiface.MetricApi, opts getMetricsOpts) ([]*providers.GenericCloudMetric, error) {
	var (
		metricChan = make(chan *providers.GenericCloudMetric, len(opts.metricsToQuery))
		errChan    = make(chan error, 1)
		doneChan   = make(chan struct{})
		metrics    = make([]*providers.GenericCloudMetric, 0)
	)
	go func() {
		waitGroup := new(sync.WaitGroup)
		for _, metric := range opts.metricsToQuery {
			waitGroup.Add(1)
			go func(metric providers.CloudProviderMetricType) {
				defer waitGroup.Done()
				metricData, err := getMetricData(ctx, metricClient, getMetricDataOpts{
					metric:    metric,
					projectID: opts.projectID,
					filter:    fmt.Sprintf(opts.filterTemplate, opts.monitoringResourceType, opts.instanceID, metric.ProviderMetricName),
					interval: &monitoringpb.TimeInterval{
						StartTime: timestamppb.New(time.Now().Add(-resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration))),
						EndTime:   timestamppb.Now(),
					},
					labels: opts.defaultLabels,
				})
				if err != nil {
					errChan <- err
				}
				metricChan <- metricData
			}(metric)
		}
		waitGroup.Wait()
		close(doneChan)
	}()
	for {
		select {
		case metricData := <-metricChan:
			metrics = append(metrics, metricData)
		case err := <-errChan:
			return nil, err
		case <-doneChan:
			return metrics, nil
		}
	}
}

type getMetricDataOpts struct {
	metric    providers.CloudProviderMetricType
	projectID string
	filter    string
	reducer   monitoringpb.Aggregation_Reducer
	interval  *monitoringpb.TimeInterval
	labels    map[string]string
}

func getMetricData(ctx context.Context, metricClient gcpiface.MetricApi, opts getMetricDataOpts) (*providers.GenericCloudMetric, error) {
	timeSeries, err := metricClient.ListTimeSeries(ctx, &monitoringpb.ListTimeSeriesRequest{
		Name:   opts.projectID,
		Filter: opts.filter,
		Aggregation: &monitoringpb.Aggregation{
			PerSeriesAligner:   monitoringpb.Aggregation_Aligner(monitoringpb.Aggregation_Aligner_value[opts.metric.Statistic]),
			AlignmentPeriod:    durationpb.New(resources.MetricsWatchDuration),
			CrossSeriesReducer: opts.reducer,
		},
		Interval: opts.interval,
		View:     monitoringpb.ListTimeSeriesRequest_FULL,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing time series: %w", err)
	}
	return &providers.GenericCloudMetric{
		Name:   opts.metric.PrometheusMetricName,
		Labels: opts.labels,
		Value:  calculatePointsAverage(timeSeries[0].Points),
	}, nil
}

func calculatePointsAverage(points []*monitoringpb.Point) float64 {
	var total float64
	for _, point := range points {
		total += point.Value.GetDoubleValue()
	}
	return total / float64(len(points))
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func calculateAvailableMemory(ctx context.Context, metricClient gcpiface.MetricApi, compoundMetric providers.CloudProviderMetricType, opts getMetricsOpts) (*providers.GenericCloudMetric, error) {
	metricPair := strings.Split(compoundMetric.ProviderMetricName, "-")
	maxMemoryMetricType := providers.CloudProviderMetricType{
		ProviderMetricName: metricPair[0],
		Statistic:          compoundMetric.Statistic,
	}
	usedMemoryMetricType := providers.CloudProviderMetricType{
		ProviderMetricName: metricPair[1],
		Statistic:          compoundMetric.Statistic,
	}
	interval := &monitoringpb.TimeInterval{
		StartTime: timestamppb.New(time.Now().Add(-resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration))),
		EndTime:   timestamppb.Now(),
	}
	maxMemoryMetric, err := getMetricData(ctx, metricClient, getMetricDataOpts{
		metric:    maxMemoryMetricType,
		projectID: opts.projectID,
		filter:    fmt.Sprintf(opts.filterTemplate, opts.monitoringResourceType, opts.instanceID, maxMemoryMetricType.ProviderMetricName),
		interval:  interval,
		labels:    opts.defaultLabels,
	})
	if err != nil {
		return nil, err
	}
	usedMemoryMetric, err := getMetricData(ctx, metricClient, getMetricDataOpts{
		metric:    usedMemoryMetricType,
		projectID: opts.projectID,
		filter:    fmt.Sprintf(opts.filterTemplate, opts.monitoringResourceType, opts.instanceID, usedMemoryMetricType.ProviderMetricName),
		interval:  interval,
		labels:    opts.defaultLabels,
	})
	if err != nil {
		return nil, err
	}
	return &providers.GenericCloudMetric{
		Name:   compoundMetric.PrometheusMetricName,
		Labels: opts.defaultLabels,
		Value:  roundFloat(maxMemoryMetric.Value-usedMemoryMetric.Value, 2),
	}, nil
}

func calculateCpuUtilization(ctx context.Context, metricClient gcpiface.MetricApi, metric providers.CloudProviderMetricType, opts getMetricsOpts) (*providers.GenericCloudMetric, error) {
	now := time.Now()
	minutesAgo := now.Add(-resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration))
	minutesAgoDoubled := now.Add(-resources.GetMetricReconcileTimeOrDefault(resources.MetricsWatchDuration * 2))
	currentSampleCpuSecs, err := getMetricData(ctx, metricClient, getMetricDataOpts{
		metric:    metric,
		projectID: opts.projectID,
		filter:    fmt.Sprintf(opts.filterTemplate, opts.monitoringResourceType, opts.instanceID, metric.ProviderMetricName),
		interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(minutesAgo),
			EndTime:   timestamppb.New(now),
		},
		reducer: monitoringpb.Aggregation_REDUCE_SUM,
		labels:  opts.defaultLabels,
	})
	if err != nil {
		return nil, err
	}
	previousSampleCpuSecs, err := getMetricData(ctx, metricClient, getMetricDataOpts{
		metric:    metric,
		projectID: opts.projectID,
		filter:    fmt.Sprintf(opts.filterTemplate, opts.monitoringResourceType, opts.instanceID, metric.ProviderMetricName),
		interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(minutesAgoDoubled),
			EndTime:   timestamppb.New(minutesAgo),
		},
		reducer: monitoringpb.Aggregation_REDUCE_SUM,
		labels:  opts.defaultLabels,
	})
	if err != nil {
		return nil, err
	}
	sampleDuration := resources.MetricsWatchDuration / time.Second
	cpuUsagePercentage := (currentSampleCpuSecs.Value - previousSampleCpuSecs.Value) / float64(sampleDuration)
	if cpuUsagePercentage < 0 {
		cpuUsagePercentage = 0
	}
	return &providers.GenericCloudMetric{
		Name:   metric.PrometheusMetricName,
		Labels: opts.defaultLabels,
		Value:  roundFloat(cpuUsagePercentage, 2),
	}, nil
}
