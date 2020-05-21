---
targets:
- 2.3.0
estimate: 30m
---

# A21 - Verify maintenance and backup windows

## AWS Infrastructure Steps
The following steps are valid for a cluster provisioned in AWS backed by AWS RDS, Elasticache and S3 resources

1. After RHMI install check for `cloud-resources-aws-strategies` config map in RHMI operator ns
    - If this config map is present make note of the values in the `CreateStrategy` for both `Redis` and `Postgres`
2. For each Resource Created in AWS RDS and Elasticache 
    - Note the maintenance and backup/snapshot windows 
3. Update the RHMIConfig maintenance `applyFrom` and backup `applyOn` fields
    - `applyOn` expects the format `HH:mm` eg. `15:04`
    - `applyFrom` expects the format `DDD HH:mm` eg. `sun 16:05`
4. Check RHMI ns for config map `cloud-resources-aws-strategies`
    - if the config map was present in step 1, ensure the `CreateStrategy` for both `Redis` and `Postgres` has updated. 
    - if the config map was not present in step 2, ensure it has been created
    - at this stage the values in `Redis CreateStrategy` should have a `snapshotWindow` that is a 1hr block starting on the `applyOn` value
    - at this stage the values in `Redis CreateStrategy` should have a `preferredMaintenanceWindow` that is a 1hr block starting on from the `applyFrom` value
    - at this stage the values in `Postgres CreateStrategy` should have a `preferredBackupWindow` that is a 1hr block starting on the `applyOn` value
    - at this stage the values in `Postgres CreateStrategy` should have a `preferredMaintenanceWindow` these is a 1hr block starting from the `applyFrom` value
5. For each Resource Created in AWS RDS and Elasticache
    - Check the maintenance and backup/snapshot windows are as expected (match the windows from step 4)
    - Note it may take several minutes for CRO to reconcile on every resource
    
    
