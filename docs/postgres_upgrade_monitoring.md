# PostgreSQL Upgrade Monitoring Implementation Guide

## Overview

This document explains how to implement monitoring for available PostgreSQL upgrades in AWS RDS instances. The implementation creates a custom metric `postgres_upgrade_available` that the RHOAM operator uses to alert when PostgreSQL upgrades are available.

## Problem Statement

AWS CloudWatch doesn't provide a direct metric for PostgreSQL upgrade availability. We need to:
1. Determine which PostgreSQL versions are available for upgrade
2. Compare current instance versions with available upgrades
3. Expose this information as a Prometheus metric
4. Create alerts based on this metric

## Implementation Options

### Option 1: Extend Cloud Resource Operator (Recommended)

Since you already use the cloud-resource-operator for PostgreSQL monitoring, extend it to include upgrade checking.

#### 1.1 Create New Metric Provider

Create a new file: `pkg/providers/aws/provider_postgres_upgrade_metrics.go`

```go
package aws

import (
    "context"
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/rds"
    "github.com/aws/aws-sdk-go/service/rds/rdsiface"
    "github.com/integr8ly/cloud-resource-operator/pkg/providers"
    "github.com/sirupsen/logrus"
    "github.com/version"
    errorUtil "github.com/pkg/errors"
)

type PostgresUpgradeMetricsProvider struct {
    Client providers.CloudResourceClient
    Logger *logrus.Entry
}

func NewAWSPostgresUpgradeMetricsProvider(client providers.CloudResourceClient, logger *logrus.Entry) (*PostgresUpgradeMetricsProvider, error) {
    return &PostgresUpgradeMetricsProvider{
        Client: client,
        Logger: logger.WithFields(logrus.Fields{"provider": "aws_postgres_upgrade_metrics"}),
    }, nil
}

func (p *PostgresUpgradeMetricsProvider) SupportsStrategy(strategy string) bool {
    return strategy == providers.AWSDeploymentStrategy
}

func (p *PostgresUpgradeMetricsProvider) ScrapeMetrics(ctx context.Context, postgres *v1alpha1.Postgres) ([]*providers.GenericCloudMetric, error) {
    logger := p.Logger.WithField("action", "ScrapeMetrics")
    
    // Get current PostgreSQL instance details
    currentVersion, err := p.getCurrentPostgresVersion(ctx, postgres)
    if err != nil {
        return nil, errorUtil.Wrap(err, "failed to get current postgres version")
    }
    
    // Get available upgrade versions
    upgradeAvailable, availableVersions, err := p.checkUpgradeAvailability(ctx, currentVersion, postgres)
    if err != nil {
        return nil, errorUtil.Wrap(err, "failed to check upgrade availability")
    }
    
    // Create metrics
    metrics := []*providers.GenericCloudMetric{}
    
    // Main upgrade availability metric
    upgradeMetric := &providers.GenericCloudMetric{
        Name:  "postgres_upgrade_available",
        Value: 0,
        Labels: map[string]string{
            "instance_id":       postgres.Status.InstanceID,
            "current_version":   currentVersion,
            "available_version": "",
        },
    }
    
    if upgradeAvailable {
        upgradeMetric.Value = 1
        if len(availableVersions) > 0 {
            upgradeMetric.Labels["available_version"] = availableVersions[0] // Latest available
        }
    }
    
    metrics = append(metrics, upgradeMetric)
    
    // Additional metrics for each available version
    for _, version := range availableVersions {
        versionMetric := &providers.GenericCloudMetric{
            Name:  "postgres_upgrade_version_available",
            Value: 1,
            Labels: map[string]string{
                "instance_id":     postgres.Status.InstanceID,
                "current_version": currentVersion,
                "target_version":  version,
            },
        }
        metrics = append(metrics, versionMetric)
    }
    
    logger.Infof("scraped postgres upgrade metrics: upgrade_available=%v, versions=%v", upgradeAvailable, availableVersions)
    return metrics, nil
}

func (p *PostgresUpgradeMetricsProvider) getCurrentPostgresVersion(ctx context.Context, postgres *v1alpha1.Postgres) (string, error) {
    sess, err := CreateSessionFromStrategy(ctx, p.Client, postgres.Spec.Tier, postgres.Spec.Type, p.Logger)
    if err != nil {
        return "", errorUtil.Wrap(err, "failed to create aws session")
    }
    
    rdsAPI := rds.New(sess)
    
    input := &rds.DescribeDBInstancesInput{
        DBInstanceIdentifier: aws.String(postgres.Status.InstanceID),
    }
    
    result, err := rdsAPI.DescribeDBInstancesWithContext(ctx, input)
    if err != nil {
        return "", errorUtil.Wrap(err, "failed to describe db instance")
    }
    
    if len(result.DBInstances) == 0 {
        return "", fmt.Errorf("no db instance found with id %s", postgres.Status.InstanceID)
    }
    
    return aws.StringValue(result.DBInstances[0].EngineVersion), nil
}

func (p *PostgresUpgradeMetricsProvider) checkUpgradeAvailability(ctx context.Context, currentVersion string, postgres *v1alpha1.Postgres) (bool, []string, error) {
    sess, err := CreateSessionFromStrategy(ctx, p.Client, postgres.Spec.Tier, postgres.Spec.Type, p.Logger)
    if err != nil {
        return false, nil, errorUtil.Wrap(err, "failed to create aws session")
    }
    
    rdsAPI := rds.New(sess)
    
    // Get available engine versions for PostgreSQL
    input := &rds.DescribeDBEngineVersionsInput{
        Engine: aws.String("postgres"),
    }
    
    result, err := rdsAPI.DescribeDBEngineVersionsWithContext(ctx, input)
    if err != nil {
        return false, nil, errorUtil.Wrap(err, "failed to describe db engine versions")
    }
    
    var availableUpgrades []string
    currentVersionParsed, err := version.NewVersion(currentVersion)
    if err != nil {
        return false, nil, errorUtil.Wrap(err, "failed to parse current version")
    }
    
    // Find current version and check its upgrade targets
    for _, engineVersion := range result.DBEngineVersions {
        if aws.StringValue(engineVersion.EngineVersion) == currentVersion {
            for _, upgrade := range engineVersion.ValidUpgradeTarget {
                targetVersion := aws.StringValue(upgrade.EngineVersion)
                targetVersionParsed, err := version.NewVersion(targetVersion)
                if err != nil {
                    p.Logger.Warnf("failed to parse target version %s: %v", targetVersion, err)
                    continue
                }
                
                // Only include newer versions
                if targetVersionParsed.GreaterThan(currentVersionParsed) {
                    availableUpgrades = append(availableUpgrades, targetVersion)
                }
            }
            break
        }
    }
    
    return len(availableUpgrades) > 0, availableUpgrades, nil
}
```

#### 1.2 Integration Point

Add the upgrade metrics provider to your existing PostgreSQL monitoring reconciliation:

```go
// In your PostgreSQL reconciler
func (r *PostgresReconciler) reconcileUpgradeMetrics(ctx context.Context, postgres *v1alpha1.Postgres) error {
    upgradeProvider, err := aws.NewAWSPostgresUpgradeMetricsProvider(r.Client, r.Logger)
    if err != nil {
        return err
    }
    
    metrics, err := upgradeProvider.ScrapeMetrics(ctx, postgres)
    if err != nil {
        r.Logger.Error("failed to scrape postgres upgrade metrics", err)
        return err
    }
    
    // Expose metrics to Prometheus
    return r.exposeMetrics(metrics)
}
```

### Option 2: Standalone Upgrade Checker

Create a separate component that runs as a CronJob or part of the operator.

#### 2.1 Create Upgrade Checker Component

```go
// pkg/postgres/upgrade_checker.go
package postgres

import (
    "context"
    "time"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/rds"
    "github.com/prometheus/client_golang/prometheus"
)

type UpgradeChecker struct {
    rdsClient *rds.RDS
    metrics   *UpgradeMetrics
}

type UpgradeMetrics struct {
    upgradeAvailable *prometheus.GaugeVec
}

func NewUpgradeChecker(sess *session.Session) *UpgradeChecker {
    metrics := &UpgradeMetrics{
        upgradeAvailable: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "postgres_upgrade_available",
                Help: "Indicates if PostgreSQL upgrade is available (1=available, 0=not available)",
            },
            []string{"instance_id", "current_version", "available_version"},
        ),
    }
    
    prometheus.MustRegister(metrics.upgradeAvailable)
    
    return &UpgradeChecker{
        rdsClient: rds.New(sess),
        metrics:   metrics,
    }
}

func (uc *UpgradeChecker) CheckUpgrades(ctx context.Context, instanceIDs []string) error {
    for _, instanceID := range instanceIDs {
        if err := uc.checkInstanceUpgrade(ctx, instanceID); err != nil {
            return fmt.Errorf("failed to check upgrade for instance %s: %w", instanceID, err)
        }
    }
    return nil
}

func (uc *UpgradeChecker) checkInstanceUpgrade(ctx context.Context, instanceID string) error {
    // Implementation similar to the provider above
    // Set metrics using uc.metrics.upgradeAvailable.WithLabelValues(...).Set(value)
}
```

## Deployment Configuration

### Option 1: Add to Existing Operator

If extending the cloud-resource-operator:

1. Add the upgrade metrics provider to the PostgreSQL reconciliation loop
2. Schedule upgrade checks every 6-24 hours (upgrades don't change frequently)
3. Use existing AWS credentials and permissions

### Option 2: Separate Component

If creating a standalone component:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-upgrade-checker
  namespace: rhoam-cloud-resources-operator
spec:
  schedule: "0 2 * * *"  # Run daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: postgres-upgrade-checker
          containers:
          - name: upgrade-checker
            image: postgres-upgrade-checker:latest
            env:
            - name: AWS_REGION
              value: "us-east-1"
            resources:
              requests:
                memory: "64Mi"
                cpu: "50m"
              limits:
                memory: "128Mi"
                cpu: "100m"
          restartPolicy: OnFailure
```

## AWS Permissions Required

Add these permissions to your existing AWS role:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "rds:DescribeDBInstances",
                "rds:DescribeDBEngineVersions"
            ],
            "Resource": "*"
        }
    ]
}
```

## Testing the Implementation

### 1. Verify Metric Exposure

```bash
# Check if metric is exposed
curl http://localhost:8080/metrics | grep postgres_upgrade_available

# Expected output:
# postgres_upgrade_available{instance_id="mydb-instance",current_version="13.7",available_version="14.6"} 1
```

### 2. Test Alert

```bash
# Create a test scenario by temporarily modifying the metric
kubectl get prometheusrule -A | grep postgres-version-updates

# Check alert firing
kubectl get prometheus -A -o yaml | grep postgres_upgrade_available
```

## Monitoring and Observability

### Grafana Dashboard

Create dashboard panels for:
- Number of instances with upgrades available
- Time since last upgrade check
- Upgrade availability by PostgreSQL version

### Alert Tuning

- **Severity**: Set to `info` since upgrades are optional
- **Frequency**: Check every 6-24 hours (upgrades don't change often)
- **Grouping**: Group by instance or cluster
- **Suppression**: Consider suppressing during maintenance windows

## Production Considerations

1. **Rate Limiting**: AWS RDS API has rate limits - don't check too frequently
2. **Error Handling**: Handle API errors gracefully and continue checking other instances
3. **Metric Retention**: Consider how long to retain upgrade availability history
4. **Multi-Region**: Ensure checks work across multiple AWS regions
5. **Cost**: Monitor AWS API call costs if checking many instances

## Next Steps

1. Choose implementation option (extending cloud-resource-operator recommended)
2. Implement the upgrade metrics provider
3. Add AWS permissions to existing role
4. Deploy and test the implementation
5. Create SOP documentation for handling upgrade alerts
6. Consider automated upgrade scheduling based on maintenance windows

## Related Documentation

- [AWS RDS API Reference](https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_DescribeDBEngineVersions.html)
- [PostgreSQL Upgrade Guide](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_UpgradeDBInstance.PostgreSQL.html)
- [Cloud Resource Operator Documentation](https://github.com/integr8ly/cloud-resource-operator) 