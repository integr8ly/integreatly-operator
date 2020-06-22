---
targets:
  - 2.5.0
---

# N06 - Verify RDS and Elasticache DB instance types are successfully migrated

## Description

This RHMI post-upgrade test case should verify that RDS and Elasticache DB instance types were successfully migrated during AWS maintenance window
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

> output should contain only "db.t2.small"

```bash
# Elasticache
aws elasticache describe-cache-clusters | jq -r '.CacheClusters[] | .CacheNodeType'
```

> output should contain only "cache.t2.micro"

3. Follow [the guide for updating the maintenance window](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/cssre_info/info_aws_update_backup_maintenance_window.md) and set it to an hour from now
4. Come back after maintenance actions are completed
5. Check updated RDS and Elasticache instance types with AWS CLI (use the same commands as before)
   > output for RDS instances should contain only "db.t3.small"
   > output for Elasticache instances should contain only "cache.t3.micro"
