---
tags:
  - happy-path
  - automated
targets:
  - 2.5.0
---

# N06 - Verify Legacy cluster VPC is configured properly

<https://github.com/integr8ly/integreatly-operator/blob/master/test/functional/aws_legacy_vpc.go>

## Description

This test case should verify that RHMI AWS resources are still configured properly after upgrade from 2.4.0
More info: <https://issues.redhat.com/browse/INTLY-8098>

## Prerequisites

- AWS console admin access to the account where the OpenShift cluster is provisioned
- [OCM CLI](https://github.com/openshift-online/ocm-cli/releases) and access to ocm organization where the cluster has been provisioned

## Steps

1. Login to OCM and get RHMI OSD cluster Infra ID and note it down

```
ocm login --url=https://api.stage.openshift.com --token=<YOUR_OCM_TOKEN>
ocm get cluster <cluster-id> | jq -r .infra_id
```

2. Go to AWS console, log in and navigate to RDS -> Databases

3. Click on some of the DB to navigate to the DB details

4. See the Networking tab

5. Click on the VPC -> tags

> Verify that the VPC belongs to the OSD cluster VPC (it should have a tag `kubernetes.io/cluster/<cluster-infra-id>`)
>
> Verify no other VPC exists in the account with a tag `integreatly.org/clusterId=<cluster-infra-id>`

6. Back in Networking tab, verify that the subnets belong to the Cluster VPC

7. Back in DB details -> Security, click on VPC security group link

> Verify that the security group belongs to the cluster VPC
