---
tags:
  - happy-path
estimate: 30m
---

# N05 - Verify upgrade not approved when another upgrade in progress

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHMI Installation

## Steps

1. Trigger an upgrade of RHMI
2. Verify that the installPlan gets approved
3. Trigger another upgrade of RHMI
4. Verify that the installPlan does not get approved until `status.stage` is `complete` and the `toVersion` field is empty.
