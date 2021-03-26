---
products:
  - name: rhmi
  - name: rhoam
tags:
  - automated
---

# A21 - Verify maintenance and backup windows

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/rhmi_config_cro_strategy_override.go

https://github.com/integr8ly/integreatly-operator/blob/master/test/functional/aws_strategy_override.go

## Steps

1. After RHMI has installed, check for `cloud-resources-aws-strategies` config map in RHMI operator ns
   - If this config map is present make note of the values in the `CreateStrategy` for both `Redis` and `Postgres`
   - Note. The config map being present at this point of the test does not affect the outcome of the test
2. For each Resource Created in AWS RDS and Elasticache
   - Take note of the maintenance and backup/snapshot windows
   - These values can be found in the AWS console:
     - Navigate to RDS, click on provisioned RDS instances, navigate to `Maintenance & Backups`
     - Navigate to Elasticache, click on `Redis`, click on each instance to unfold the menu, this will show `snapshot` and `maintenance` times
3. Update the RHMIConfig maintenance `applyFrom` and backup `applyOn` fields
   - `applyOn` expects the format `HH:mm` eg. `15:04`
   - `applyFrom` expects the format `DDD HH:mm` eg. `sun 16:05`
4. Check RHMI ns for config map `cloud-resources-aws-strategies`
   - Ensure the ConfigMap `cloud-resources-aws-strategies` now exists and has the following elements:
     - `Redis CreateStrategy` and `Postgres CreateStrategy` should have a `snapshotWindow` that is a 1hr block starting on the `applyOn` value
     - `Postgres CreateStrategy` should have a `preferredBackUpWindow` that is a 1hr block starting on the `applyOn` value
     - `Redis CreateStrategy` and `Postgres CreateStrategy` should have a `preferredMaintenanceWindow` that is a 1hr block starting on from the `applyFrom` value
5. For each Resource Created in AWS RDS and Elasticache
   - Check the maintenance and backup/snapshot windows are as expected (match the windows from step 4, and differ from the windows in step 2)
   - **Note**: it may take several minutes for the Cloud Resource Operator (CRO) to reconcile on every resource
   - These values can be found in the AWS console:
     - Navigate to RDS, click on provisioned RDS instances, navigate to `Maintenance & Backups`
     - Navigate to Elasticache, click on `Redis`, click on each instance to unfold the menu, this will show `snapshot` and `maintenance` times
