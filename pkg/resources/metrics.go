// utility functions for creating and reconciling on cloud resource alerts
//
// alerts created :
//  * Postgres Availability Alerts (per product)
//  * Postgres Connectivity Alerts (per product)
//  * Postgres Resource Status Phase Pending (per product)
//  * Postgres Resource Status Phase Failed (per product)
//  * Redis Availability Alerts (per product)
//  * Redis Connectivity Alerts (per product)
//  * Redis Resource Status Phase Pending (per product)
//  * Redis Resource Status Phase Failed (per product)
//  * Redis will run out of memory in 4 days (per 3scale redis)
//  * Redis will run out of memory in 4 hours (per 3scale redis)
//  * Redis high memory usage for the last hour (per 3scale redis)
//  * Postgres will run out of space in 4 days (per product)
//  * Postgres will run out of space in 4 hours (per product)

package resources

import (
	"context"
	"fmt"
	"strings"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	cro1types "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// TODO: these Metric names should be imported from github.com/integr8ly/cloud-resource-operator/pkg/resources v0.18.0 once it is possible to update to that version
	DefaultPostgresDeletionMetricName = "cro_postgres_deletion_timestamp"
	DefaultRedisDeletionMetricName    = "cro_redis_deletion_timestamp"
	alertFor5Mins                     = "5m"
	alertFor15Mins                    = "15m"
	alertFor10Mins                    = "10m"
	alertFor20Mins                    = "20m"
	alertFor30Mins                    = "30m"
	alertFor60Mins                    = "60m"
	alertPercentage                   = "90"
)

func ReconcilePostgresAlerts(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger) (v1alpha1.StatusPhase, error) {
	// create prometheus failed rule
	_, err := createPostgresResourceStatusPhaseFailedAlert(ctx, client, inst, cr, log, inst.Spec.Type)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres failure alert for %s: %w", cr.Name, err)
	}

	// create the prometheus deletion rule
	if _, err = createPostgresResourceDeletionStatusFailedAlert(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres deletion prometheus alert for %s: %w", cr.Name, err)
	}

	if cr.Status.Phase != cro1types.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// create the prometheus pending rule
	_, err = createPostgresResourceStatusPhasePendingAlert(ctx, client, inst, cr, log, inst.Spec.Type)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres pending alert for %s: %w", cr.Name, err)
	}

	// create the prometheus availability rule
	if _, err = createPostgresAvailabilityAlert(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus alert for %s: %w", cr.Name, err)
	}

	// create the prometheus connectivity rule
	if _, err = createPostgresConnectivityAlert(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres connectivity prometheus alert for %s: %w", cr.Name, err)
	}

	// create the prometheus deletion rule
	if _, err = createPostgresResourceDeletionStatusFailedAlert(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres deletion prometheus alert for %s: %w", cr.Name, err)
	}

	// create the prometheus free storage alert rules
	if err = reconcilePostgresFreeStorageAlerts(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres free storage prometheus alerts for %s: %w", cr.Name, err)
	}

	if err = reconcilePostgresFreeableMemoryAlert(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres freeable memory alert for %s: %w", cr.Name, err)
	}

	// create the prometheus high cpu alert rule
	if err = reconcilePostgresCPUUtilizationAlerts(ctx, client, inst, cr, log, inst.Spec.Type); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres cpu utilization prometheus alerts for %s: %w", cr.Name, err)
	}

	return v1alpha1.PhaseCompleted, nil
}

func ReconcileRedisAlerts(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) (v1alpha1.StatusPhase, error) {

	// redis cr returning a failed state
	_, err := createRedisResourceStatusPhaseFailedAlert(ctx, client, inst, cr, log)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis failure alert %s: %w", cr.Name, err)
	}

	// redis cr returning a failed state during deletion
	_, err = createRedisResourceDeletionStatusFailedAlert(ctx, client, inst, cr, log)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis deletion failure alert for %s: %w", cr.Name, err)
	}

	if cr.Status.Phase != cro1types.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// create prometheus pending rule
	_, err = createRedisResourceStatusPhasePendingAlert(ctx, client, inst, cr, log)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis pending alert %s: %w", cr.Name, err)
	}

	// create the prometheus availability rule
	_, err = createRedisAvailabilityAlert(ctx, client, inst, cr, log)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis prometheus alert for %s: %w", cr.Name, err)
	}
	// create backend connectivity alert
	_, err = createRedisConnectivityAlert(ctx, client, inst, cr, log)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis prometheus connectivity alert for %s: %w", cr.Name, err)
	}

	// create Redis Memory Usage High alert
	if err = createRedisMemoryUsageAlerts(ctx, client, inst, cr, log); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis prometheus memory usage high alerts for %s: %w", cr.Name, err)
	}

	// create Redis Cpu Usage High Alert
	if err = CreateRedisCpuUsageAlerts(ctx, client, inst, cr, log); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create redis prometheus cpu usage high alerts for %s: %w", cr.Name, err)
	}

	return v1alpha1.PhaseCompleted, nil
}

// CreateSmtpSecretExists creates a PrometheusRule to alert if the rhmi-smtp-secret is present
// the ocm sendgrid service creates a secret automatically this is a check for when that service fails
func CreateSmtpSecretExists(ctx context.Context, client k8sclient.Client, cr *v1alpha1.RHMI) (v1alpha1.StatusPhase, error) {
	installationName := InstallationNames[cr.Spec.Type]

	alertName := "SendgridSmtpSecretExists"
	ruleName := "sendgrid-smtp-secret-exists-rule"
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(kube_secret_info{namespace='%s',secret='"+cr.Spec.NamespacePrefix+"smtp'} == 1)", cr.Namespace),
	)
	alertDescription := fmt.Sprintf("The Sendgrid SMTP secret has not been created in the %s namespace and may need to be created manualy", cr.Namespace)
	labels := map[string]string{
		"severity": "warning",
		"product":  installationName,
	}
	// create the rule
	_, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlSendGridSmtpSecretExists, alertFor10Mins, alertExp, labels)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create sendgrid smtp exists rule err: %s", err)
	}
	return v1alpha1.PhaseCompleted, nil
}

// createPostgresAvailabilityAlert creates a PrometheusRule alert to watch for the availability
// of a Postgres instance
func createPostgresAvailabilityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) (*prometheusv1.PrometheusRule, error) {
	installationName := InstallationNames[installType]

	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres alert creation, useClusterStorage is true")
		return nil, nil
	}

	productName := cr.Labels["productName"]
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresInstanceUnavailable"
	sopURL := sopUrlRhoamBase + alertName + ".asciidoc"
	alertSeverity := "critical"
	if strings.Contains(productName, "sso") {
		// Setting alert severity level to warning for Cluster and User SSO redis alerts as we don't want to
		// trigger a Pagerduty incident for Rate Limiting. Will need to revisit Post GA.
		// https://issues.redhat.com/browse/MGDAPI-587
		alertSeverity = "warning"
		sopURL = sopUrlPostgresInstanceUnavailable
	}
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultPostgresAvailMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Postgres instance: '%s' (strategy: %s) for product: %s is unavailable", cr.Name, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    alertSeverity,
		"productName": cr.Labels["productName"],
		"product":     installationName,
	}

	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopURL, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createPostgresConnectivityAlert creates a PrometheusRule alert to watch for the connectivity
// of a Postgres instance
func createPostgresConnectivityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) (*prometheusv1.PrometheusRule, error) {
	installationName := InstallationNames[installType]

	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres connectivity alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresConnectionFailed"
	sopURL := sopUrlRhoamBase + alertName + ".asciidoc"
	alertSeverity := "critical"
	if strings.Contains(productName, "sso") {
		// Setting alert severity level to warning for Cluster and User SSO redis alerts as we don't want to
		// trigger a Pagerduty incident for Rate Limiting. Will need to revisit Post GA.
		// https://issues.redhat.com/browse/MGDAPI-587
		alertSeverity = "warning"
		sopURL = sopUrlPostgresConnectionFailed
	}
	ruleName := fmt.Sprintf("connectivity-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultPostgresConnectionMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Unable to connect to Postgres instance. Postgres Custom Resource: %s in namespace %s (strategy: %s) for product: %s", cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    alertSeverity,
		"productName": cr.Labels["productName"],
		"product":     installationName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopURL, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createPostgresResourceStatusPhasePendingAlert creates a PrometheusRule alert to watch for Postgres CR state
func createPostgresResourceStatusPhasePendingAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) (*prometheusv1.PrometheusRule, error) {
	installationName := InstallationNames[installType]

	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres state alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresResourceStatusPhasePending"
	ruleName := fmt.Sprintf("resource-status-phase-pending-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s',statusPhase='complete'} == 1)",
			croResources.DefaultPostgresStatusMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("The creation of the Postgres instance has take longer that %s. Postgres Custom Resource: %s in namespace %s (strategy: %s) for product: %s", alertFor20Mins, cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
		"product":     installationName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlPostgresResourceStatusPhasePending, alertFor20Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createPostgresResourceStatusPhaseFailedAlert creates a PrometheusRule alert to watch for Postgres CR state
func createPostgresResourceStatusPhaseFailedAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) (*prometheusv1.PrometheusRule, error) {
	installationName := InstallationNames[installType]

	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres state alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresResourceStatusPhaseFailed"
	ruleName := fmt.Sprintf("resource-status-phase-failed-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("(%s{exported_namespace='%s',resourceID='%s',productName='%s',statusPhase='failed'}) == 1 ",
			croResources.DefaultPostgresStatusMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("The creation of the Postgres instance has been failing longer that %s. Postgres Custom Resource: %s in namespace %s (strategy: %s) for product: %s", alertFor5Mins, cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
		"product":     installationName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlPostgresResourceStatusPhaseFailed, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createPostgresResourceDeletionStatusFailedAlert creates a PrometheusRule alert that watches for failed deletions of Postgres CRs
func createPostgresResourceDeletionStatusFailedAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) (*prometheusv1.PrometheusRule, error) {
	installationName := InstallationNames[installType]

	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres state alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresResourceDeletionStatusPhaseFailed"
	ruleName := fmt.Sprintf("resource-deletion-status-phase-failed-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("%s{exported_namespace='%s',resourceID='%s',productName='%s',statusPhase='failed'}", DefaultPostgresDeletionMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("The deletion of the Postgres instance has been failing longer than %s. Postgres Custom Resource: %s in namespace %s (strategy: %s) for product: %s", alertFor5Mins, cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
		"product":     installationName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlCloudResourceDeletionStatusFailed, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// reconcilePostgresFreeStorageAlerts reconciles on both free storage alerts (4 days and 4 hours) and a low storage alert
// To avoid any false positives when the instances are being deployed for linear projection (4 days and 4 hours)
// the alert query requires a minimum time of data before it will evaluate if the instance would run out of storage.
//
// the low storage alert fires if storage is under 10% of current capacity, with a 30 minute alertOn value to allow for any
// provider autoscaling to happen, if after 30 minutes the instance will require manual intervention
func reconcilePostgresFreeStorageAlerts(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) error {
	installationName := InstallationNames[installType]

	// dont create the alert if we are using in cluster storage
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres free storage alert creation, useClusterStorage is true")
		return nil
	}

	// job to check time that the operator metrics are exposed
	job := "cloud-resource-operator-metrics"

	// build and reconcile postgres will fill in 4 hours alert
	alertName := "PostgresStorageWillFillIn4Hours"
	sopURL := sopUrlRhoamBase + alertName + ".asciidoc"
	ruleName := "postgres-storage-will-fill-in-4-hours"
	alertDescription := "The postgres instance {{ $labels.instanceID }} for product {{  $labels.productName  }} will run of disk space in the next 4 hours"
	labels := map[string]string{
		"severity": "critical",
		"product":  installationName,
	}

	// building a predict_linear query using 2 hour of data points to predict a 4 hour projection, and checking if it is less than or equal 0
	//    * [1h] - one hour data points
	//    * , 5 * 3600 - multiplying data points by 5 hour, to allow 1 hour of pending before firing the alert
	alertExp := intstr.FromString(
		fmt.Sprintf("(predict_linear(sum by (instanceID) (cro_postgres_free_storage_average{job='%s'})[1h:1m], 5 * 3600) <= 0 and on (instanceID) (cro_postgres_free_storage_average < ((cro_postgres_current_allocated_storage / 100) * 25)))", job))

	_, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopURL, alertFor60Mins, alertExp, labels)
	if err != nil {
		return err
	}

	// build and reconcile postgres will fill in 4 days alert
	alertName = "PostgresStorageWillFillIn4Days"
	ruleName = "postgres-storage-will-fill-in-4-days"
	alertDescription = "The postgres instance {{ $labels.instanceID }} for product {{  $labels.productName  }} will run of disk space in the next 4 days"
	labels = map[string]string{
		"severity": "warning",
		"product":  installationName,
	}

	// building a predict_linear query using 2 hour of data points to predict a 4 day projection, and checking if it is less than or equal 0
	//    * [2h] - 2 hour data points
	//    * , 4 * 24 * 3600 - multiplying data points by 4 days
	alertExp = intstr.FromString(
		fmt.Sprintf("(predict_linear(sum by (instanceID) (cro_postgres_free_storage_average{job='%s'})[6h:1m], 4 * 24 * 3600) <= 0) and on (instanceID) (cro_postgres_free_storage_average < ((cro_postgres_current_allocated_storage / 100) * 25))", job))

	_, err = reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlPostgresWillFill, alertFor60Mins, alertExp, labels)
	if err != nil {
		return err
	}

	// build and reconcile postgres low storage alert
	alertName = "PostgresStorageLow"
	ruleName = "postgres-storage-low"
	alertDescription = "The postgres instance {{ $labels.instanceID }} for product {{  $labels.productName  }}, storage is currently under 10 percent of its capacity"
	labels = map[string]string{
		"severity": "warning",
		"product":  installationName,
	}

	// checking if the percentage of free storage is less than 10% of the current allocated storage
	alertExp = intstr.FromString("cro_postgres_free_storage_average < ((cro_postgres_current_allocated_storage / 100 ) * 10)")

	_, err = reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlPostgresWillFill, alertFor30Mins, alertExp, labels)
	if err != nil {
		return err
	}
	return nil
}

func reconcilePostgresFreeableMemoryAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) error {
	installationName := InstallationNames[installType]

	// dont create the alert if we are using in cluster storage
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres free storage alert creation, useClusterStorage is true")
		return nil
	}

	// build and reconcile postgres low freeable memory alert
	alertName := "PostgresFreeableMemoryLow"
	ruleName := "postgres-freeable-memory-low"
	alertDescription := "The postgres instance {{ $labels.instanceID }} for product {{  $labels.productName  }}, freeable memory is currently under 10 percent of its capacity"
	labels := map[string]string{
		"severity": "warning",
		"product":  installationName,
	}

	// checking if the percentage of freeable memory is less than 10% of the max memory
	// cro_postgres_max_memory is in MiB so cro_postgres_freeable_memory_average needs to be converted from bytes to MiB
	// conversion formula is MiB = bytes / (1024^2)
	alertExp := intstr.FromString("(cro_postgres_freeable_memory_average / (1024*1024)) < ((cro_postgres_max_memory / 100 ) * 10)")

	_, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlPostgresFreeableMemoryLow, alertFor5Mins, alertExp, labels)
	if err != nil {
		return err
	}
	return nil
}

func reconcilePostgresCPUUtilizationAlerts(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres, log l.Logger, installType string) error {
	installationName := InstallationNames[installType]

	// dont create the alert if we are using in cluster storage
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping postgres free storage alert creation, useClusterStorage is true")
		return nil
	}

	alertName := "PostgresCPUHigh"
	ruleName := "postgres-cpu-high"
	alertDescription := "the postgres instance {{ $labels.instanceID }} for product {{ $labels.productName }} has been using {{ $value }}% of available CPU for 15 minutes or more"
	labels := map[string]string{
		"severity": "warning",
		"product":  installationName,
	}

	alertExp := intstr.FromString("cro_postgres_cpu_utilization_average > 90")

	_, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlPostgresCpuUsageHigh, alertFor15Mins, alertExp, labels)
	if err != nil {
		return err
	}
	return nil
}

// createRedisResourceStatusPhasePendingAlert creates a PrometheusRule alert to watch for Redis CR state
func createRedisResourceStatusPhasePendingAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisResourceStatusPhasePending"
	ruleName := fmt.Sprintf("resource-status-phase-pending-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s',statusPhase='complete'} == 1)",
			croResources.DefaultRedisStatusMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("The creation of the Redis cache has take longer that %s. Redis Custom Resource: %s in namespace %s (strategy: %s) for product: %s", alertFor20Mins, cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlRedisResourceStatusPhasePending, alertFor20Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// CreateRedisMemoryUsageHighAlert creates a PrometheusRule alert to watch for High Memory usage
// of a Redis cache
func createRedisMemoryUsageAlerts(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) error {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis memory usage high alert creation, useClusterStorage is true")
		return nil
	}
	productName := cr.Labels["productName"]

	alertName := "RedisMemoryUsageHigh"
	ruleName := fmt.Sprintf("redis-memory-usage-high")
	alertDescription := "Redis Memory for instance {{ $labels.instanceID }} is 90 percent or higher for the last hour. Redis Custom Resource: {{ $labels.resourceID }} in namespace {{ $labels.namespace }} for the product: {{ $labels.productName }}"
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
	}

	alertExp := intstr.FromString(fmt.Sprintf("cro_redis_memory_usage_percentage_average > %s", alertPercentage))

	_, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlRedisMemoryUsageHigh, alertFor60Mins, alertExp, labels)
	if err != nil {
		return err
	}

	// job to check time that the operator metrics are exposed
	job := "cloud-resource-operator-metrics"

	alertName = "RedisMemoryUsageMaxIn4Hours"
	ruleName = fmt.Sprintf("redis-memory-usage-will-max-in-4-hours")
	alertDescription = "Redis Memory Usage is predicted to max with in four hours for instance {{ $labels.instanceID }}. Redis Custom Resource: {{ $labels.resourceID }} in namespace {{ $labels.namespace }} for the product: {{ $labels.productName }}"
	labels = map[string]string{
		"severity":    "warning",
		"productName": productName,
	}
	// building a predict_linear query using 1 hour of data points to predict a 4 hour projection, and checking if it is less than or equal 0
	//    * [1h] - one hour data points
	//    * , 4 * 3600 - multiplying data points by 4 hours
	alertExp = intstr.FromString(fmt.Sprintf("(predict_linear(sum by (instanceID) (cro_redis_memory_usage_percentage_average{job='%s'})[1h:1m], 5 * 3600) >= 100) and on (instanceID) (cro_redis_memory_usage_percentage_average{job='%s'} > 75)", job, job))

	_, err = reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlRedisMemoryUsageHigh, alertFor60Mins, alertExp, labels)
	if err != nil {
		return err
	}

	alertName = "RedisMemoryUsageMaxIn4Days"
	ruleName = fmt.Sprintf("redis-memory-usage-max-fill-in-4-days")
	alertDescription = "Redis Memory Usage is predicted to max in four days for instance {{ $labels.instanceID }}. Redis Custom Resource: {{ $labels.resourceID }} in namespace {{ $labels.namespace }} for the product: {{ $labels.productName }}"
	labels = map[string]string{
		"severity":    "warning",
		"productName": productName,
	}
	// building a predict_linear query using 1 hour of data points to predict a 4 hour projection, and checking if it is less than or equal 0
	//    * [6h] - six hour data points
	//    * , 4 * 24 * 3600 - multiplying data points by 4 days
	alertExp = intstr.FromString(fmt.Sprintf("(predict_linear(sum by (instanceID) (cro_redis_memory_usage_percentage_average{job='%s'})[6h:1m], 4 * 24 * 3600) >= 100) and on (instanceID) (cro_redis_memory_usage_percentage_average{job='%s'} > 75)", job, job))

	_, err = reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlRedisMemoryUsageHigh, alertFor60Mins, alertExp, labels)
	if err != nil {
		return err
	}

	return nil
}

// createRedisResourceStatusPhaseFailedAlert creates a PrometheusRule alert to watch for Redis CR state
func createRedisResourceStatusPhaseFailedAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisResourceStatusPhaseFailed"
	ruleName := fmt.Sprintf("resource-status-phase-failed-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("(%s{exported_namespace='%s',resourceID='%s',productName='%s',statusPhase='failed'}) == 1 ",
			croResources.DefaultRedisStatusMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("The creation of the Redis cache is Failing longer that %s. Redis Custom Resource: %s in namespace %s (strategy: %s) for product: %s", alertFor5Mins, cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlRedisResourceStatusPhaseFailed, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createRedisResourceDeletionStatusFailedAlert creates a PrometheusRule alert that watches for failed deletions of Redis CRs
func createRedisResourceDeletionStatusFailedAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis state alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisResourceDeletionStatusPhaseFailed"
	ruleName := fmt.Sprintf("resource-deletion-status-phase-failed-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("%s{exported_namespace='%s',resourceID='%s',productName='%s',statusPhase='failed'}", DefaultRedisDeletionMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("The deletion of the Redis instance has been failing longer than %s. Redis Custom Resource: %s in namespace %s (strategy: %s) for product: %s", alertFor5Mins, cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlCloudResourceDeletionStatusFailed, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createRedisAvailabilityAlert creates a PrometheusRule alert to watch for the availability
// of a Redis cache
func createRedisAvailabilityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisCacheUnavailable"
	sopURL := sopUrlRhoamBase + alertName + ".asciidoc"
	alertSeverity := "critical"
	if productName == "marin3r" {
		// Setting alert severity level to warning for Marin3r redis alerts as we don't want to
		// trigger a Pagerduty incident for Rate Limiting. Will need to revisit Post GA.
		// https://issues.redhat.com/browse/MGDAPI-587
		alertSeverity = "warning"
		sopURL = sopUrlRedisCacheUnavailable
	}
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultRedisAvailMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Redis instance: '%s' (strategy: %s) for the product: %s is unavailable", cr.Name, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    alertSeverity,
		"productName": productName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopURL, alertFor5Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// createRedisConnectivityAlert creates a PrometheusRule alert to watch for the connectivity
// of a Redis cache
func createRedisConnectivityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis connectivity alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisCacheConnectionFailed"
	sopURL := sopUrlRhoamBase + alertName + ".asciidoc"
	alertSeverity := "critical"
	if productName == "marin3r" {
		// Setting alert severity level to warning for Marin3r redis alerts as we don't want to
		// trigger a Pagerduty incident for Rate Limiting. Will need to revisit Post GA.
		// https://issues.redhat.com/browse/MGDAPI-587
		alertSeverity = "warning"
		sopURL = sopUrlRedisConnectionFailed
	}
	ruleName := fmt.Sprintf("connectivity-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultRedisConnectionMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Unable to connect to Redis instance. Redis Custom Resource: %s in namespace %s (strategy: %s) for the product: %s", cr.Name, cr.Namespace, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    alertSeverity,
		"productName": productName,
	}
	// create the rule
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopURL, alertFor60Mins, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// CreateRedisCpuUsageAlerts creates a PrometheusRule alerts to watch for High Cpu usage
// of a Redis cache
func CreateRedisCpuUsageAlerts(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis, log l.Logger) error {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		log.Info("skipping redis memory usage high alert creation, useClusterStorage is true")
		return nil
	}
	productName := cr.Labels["productName"]
	alertName := "RedisCpuUsageHigh"
	ruleName := fmt.Sprintf("redis-cpu-usage-high")
	alertDescription := "Redis Cpu for instance {{ $labels.instanceID }} is 90 percent or higher for the last hour. Redis Custom Resource: {{ $labels.resourceID }} in namespace {{ $labels.namespace }} for the product: {{ $labels.productName }}"
	labels := map[string]string{
		"severity":    "warning",
		"productName": productName,
	}

	alertExp := intstr.FromString(fmt.Sprintf("cro_redis_engine_cpu_utilization_average > %s", alertPercentage))

	_, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, sopUrlRedisCpuUsageHigh, alertFor15Mins, alertExp, labels)
	if err != nil {
		return err
	}
	return nil
}

// reconcilePrometheusRule will create a PrometheusRule object
func reconcilePrometheusRule(ctx context.Context, client k8sclient.Client, ruleName, ns, alertName, desc, sopURL, alertFor string, alertExp intstr.IntOrString, labels map[string]string) (*prometheusv1.PrometheusRule, error) {
	alertGroupName := alertName + "Group"
	groups := []prometheusv1.RuleGroup{
		{
			Name: alertGroupName,
			Rules: []prometheusv1.Rule{
				{
					Alert:  alertName,
					Expr:   alertExp,
					For:    alertFor,
					Labels: labels,
					Annotations: map[string]string{
						"description": desc,
						"sop_url":     sopURL,
					},
				},
			},
		},
	}

	rule := &prometheusv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: ns,
			Labels: map[string]string{
				"monitoring-key": "middleware",
			},
		},
		Spec: prometheusv1.PrometheusRuleSpec{
			Groups: groups,
		},
	}

	// create or update the resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.Name = ruleName
		rule.Namespace = ns
		rule.Spec.Groups = []prometheusv1.RuleGroup{
			{
				Name: alertGroupName,
				Rules: []prometheusv1.Rule{
					{
						Alert:  alertName,
						Expr:   alertExp,
						For:    alertFor,
						Labels: labels,
						Annotations: map[string]string{
							"description": desc,
							"sop_url":     sopURL,
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile prometheus rule request for %s", ruleName)
	}

	return rule, nil
}
