---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
estimate: 30m
tags:
  - destructive
---

# O02B - Verify if CRO operator removes remaining snapshots for RDS, Elasticache and S3 after the deletion of RHMI operator has finished

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster (ideally a cluster that is about to be deleted)

## Description

By following this test case should be able to verify:

- There are no manual or automatic RDS, Elasticache and S3 snapshots remaining in the AWS account

## Prerequisites

- admin access to the AWS account where the OpenShift cluster is provisioned (IAM access key & secret)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed locally
- cluster-admin (kubeadmin) access to the OpenShift instance used for verification

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```bash
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Trigger RHOAM uninstallation

```bash
oc delete rhmi rhmi -n redhat-managed-api-operator
```

3. Verify the `cloud-resources-aws-strategies` configmap in the rhmi operator has got the fields:

   - `blobstorage.production.deleteStrategy.forceBucketDeletion` set to true.
   - `postgres.production.deleteStrategy.SkipFinalSnapshot` set to true.
   - `redis.production.deleteStrategy.FinalSnapshotIdentifier` set to empty.

```bash
oc describe configmap cloud-resources-aws-strategies -n redhat-managed-api-operator
```

4. Ensure that there are no manual or automatic RDS snapshots associated to the install remaining in the AWS account.

```bash
for id in $(oc get postgres -n redhat-managed-api-operator -o json | jq -r ".items[].metadata.annotations.resourceIdentifier");
do
  if [[ -n $(aws rds describe-db-snapshots --db-instance-identifier=$id | jq -r '.DBSnapshots[]') ]]
  then
    echo "still postgres snapshots remaining";
    break;
  fi
done
```

> should return empty.

5. Ensure that there are no manual or automatic Elasticache snapshots associated to the install remaining in the AWS account.

```bash
for id in $(oc get redis -n redhat-managed-api-operator -o json | jq -r ".items[].metadata.annotations.resourceIdentifier");
do
  if [[ -n $(aws elasticache describe-snapshots --cache-cluster-id=$id-001 | jq -r '.Snapshots[]') ]]
  then
    echo "still snapshots remaining";
    break;
  fi
done
```

> should return empty.

6. Ensure that there are no S3 buckets for the cluster remaining in the AWS account.

```bash
aws_s3_buckets=$(aws s3 ls)

for id in $(oc get blobstorages -n redhat-managed-api-operator -o json | jq -r ".items[].metadata.annotations.resourceIdentifier");
  do echo $aws_s3_buckets | grep $b || echo bucket $id not found ;
done
```

> should return `bucket not found` twice.
