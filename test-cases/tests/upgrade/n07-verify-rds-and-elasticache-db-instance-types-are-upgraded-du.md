---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.5.0
---

# N07 - Verify RDS and Elasticache DB instance types are upgraded during maintenance

**_Note: This upgrade test is only relevant for clusters upgrading from RHMI 2.4.0_**

## Description

This RHMI test case should verify that maintenance for RDS and Elasticache instance types upgrades in AWS was
successfully setup by the Cloud Resource Operator. Instance types should be migrated from `t2` instance types to `t3`
instance types.

The action taken in this test is to force AWS maintenance as soon as possible, to confirm the maintenance has the
expected results.

More info: <https://issues.redhat.com/browse/INTLY-8447>

## Prerequisites

- admin access to the AWS account where the OpenShift cluster is provisioned (IAM access key & secret)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed locally
- make sure the RHMI is already upgraded to version 2.5.0

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Check the RDS and Elasticache instance types with AWS CLI:

```bash
# RDS
aws rds describe-db-instances | jq -r '.DBInstances[] | .DBInstanceClass'
```

> output should contain only 6 "db.t2.small" instances. if there are more, check for multiple clusters in the testing
> AWS account.

```bash
# Elasticache
aws elasticache describe-cache-clusters | jq -r '.CacheClusters[] | .CacheNodeType'
```

> output should contain only 2 "cache.t2.micro" instances. if there are more, check for multiple clusters in the
> testing AWS account

3. Follow [the guide for updating the maintenance window](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/cssre_info/info_aws_update_backup_maintenance_window.md) and set it to an hour from now
4. Wait for the maintenance window to be completed in AWS
5. Check updated RDS and Elasticache instance types with AWS CLI (use the same commands as before)
   > output for RDS instances should contain only "db.t3.small"
   > output for Elasticache instances should contain only "cache.t3.micro"
6. Verify the correct Redis/Elasticache engine version:

```bash
aws elasticache describe-cache-clusters | jq -r '.CacheClusters[].EngineVersion'
```

> output should contain only the versions "5.0.6"
