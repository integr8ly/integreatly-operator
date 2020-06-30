---
tags:
  - happy-path
  - automated
targets:
  - 2.5.0
---

# A25 - Verify standalone RHMI VPC exists and is configured properly

<https://github.com/integr8ly/integreatly-operator/blob/master/test/functional/aws_standalone_vpc.go>

## Description

This test case should verify that standalone VPC was created in AWS and it is configured properly
More info: <https://issues.redhat.com/browse/INTLY-8098>

It is not valid for RHMI clusters upgraded from previous versions (<2.5.0)

## Prerequisites

- admin access to the AWS account where the OpenShift cluster is provisioned (IAM access key & secret)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed locally
- [OCM CLI](https://github.com/openshift-online/ocm-cli/releases) and access to ocm organization where the cluster has been provisioned

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Note down CIDR block from `cloud-resources-aws-strategies` config map (search for `_network` field)

```
oc get configmap cloud-resources-aws-strategies -n redhat-rhmi-operator -o yaml
```

3. Get OSD cluster Infra ID and note it down

```
CID=$(ocm get cluster <cluster-id> | jq -r .infra_id)
```

4. List the VPCs in AWS account

```
aws ec2 describe-vpcs --filter "Name=tag-value,Values=$CID"
```

> Verify that the VPC was created and verify that the CIDR block is the same as the one previously noted

5. Following list of verification steps are effectively covered by the functional test
   > Standalone VPC subnet exists and has the expected CIRD block
   > Subnet masks for the subnets are one bit greater than the VPC subnet mask
   > Single VPC security groups with the correct tag exists
   > All RDS subnets exist in a single subnet group
   > All Elasticache subnets exist in a single subnet group
   > Single VPC peering connection with the correct tag exists and is active
   > VPC route table with the correct tag exists and has a route to the peering connection
   > Cluster route tables with the correct tag exist and contain a route to the peering connection and the standalone vpc
