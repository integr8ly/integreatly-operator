---
products:
  - name: rhmi
    environments:
      - external
estimate: 30m
tags:
  - manual-selection
---

# N05a - Verify upgrade not approved when another upgrade in progress

## Description

**Note:** This test can be executed only when installing RHMI using olm, just ignore it when testing RHMI on OSD

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
