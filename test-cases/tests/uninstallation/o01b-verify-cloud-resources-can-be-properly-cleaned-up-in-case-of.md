---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
estimate: 30m
tags:
  - destructive
---

# O01b - Verify cloud resources can be properly cleaned up in case of failing RHOAM uninstallation

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster (ideally a cluster that is about to be deleted)

## Description

By following this test case should be able to verify:

- alerts are triggered when RHOAM Operator is unable to remove cloud resources during "uninstallation" phase
- SOP for Failed Automatic Teardown of Cloud Resources

## Prerequisites

- admin access to the AWS account where the OpenShift cluster is provisioned (IAM access key & secret)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed locally
- [OCM CLI](https://github.com/openshift-online/ocm-cli/releases) and access to ocm organization where the cluster has been provisioned
- cluster-admin (kubeadmin) access to the OpenShift instance used for verification

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```bash
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Verify that new alerts for indicating issues with deleting cloud resources are present:

```bash
oc get prometheusrule -n redhat-managed-api-operator | grep -cE "resource-deletion((.*rhsso|.*rhssouser|.*threescale)-postgres|(.*threescale|.*threescale-backend)-redis)"
```

> You should get "8" in the output

3. Patch the `cloud-resources-aws-strategies` config map with a dummy value in `region` field for `postgres` and `redis` instances

```bash
postgres=$(oc get configmap cloud-resources-aws-strategies -n redhat-managed-api-operator -o jsonpath='{.data.postgres}' | jq -c '.production.region = "blabla123"' | jq -R)
redis=$(oc get configmap cloud-resources-aws-strategies -n redhat-managed-api-operator -o jsonpath='{.data.redis}' | jq -c '.production.region = "blabla123"' | jq -R)
oc patch configmap cloud-resources-aws-strategies -n redhat-managed-api-operator --type=merge --patch="{\"data\": { \"postgres\": $postgres }}" --dry-run=false
oc patch configmap cloud-resources-aws-strategies -n redhat-managed-api-operator --type=merge --patch="{\"data\": { \"redis\": $redis }}" --dry-run=false
```

4. Trigger RHOAM uninstallation

```bash
oc delete rhmi rhoam -n redhat-managed-api-operator
```

5. Trigger Redis CR deletion (Note: this step won't be necessary after https://issues.redhat.com/browse/INTLY-9101 is resolved)

```bash
oc delete redis --all -n redhat-managed-api-operator
```

6. Go to alert manager

```bash
open "https://$(oc get routes alertmanager-route -n redhat-managed-api-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
```

> Verify that all Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed and Redis-RhmiRedisResourceDeletionStatusPhaseFailed alerts (8 in total) go into a pending state and then they start firing (it should take 5 minutes for these alerts to go from pending to firing state)

7. Verify [this SOP](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/uninstall/delete_cluster_teardown.md#procedure) (guide to delete the cluster and related RHMI Cloud Resources)
