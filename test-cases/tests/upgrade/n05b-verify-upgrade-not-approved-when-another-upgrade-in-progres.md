---
products:
  - name: rhoam
    environments:
      - external
estimate: 30m
tags:
  - manual-selection
---

# N05B - Verify upgrade not approved when another upgrade in progress

## Description

Obsolete - this is a very edge case and can't be tested via addon flow.

**Note:** This test can be executed only when installing RHOAM using olm, just ignore it when testing RHOAM on OSD

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM Installation

## Steps

1. Trigger an upgrade of RHOAM
2. Verify that the installPlan gets approved
3. Trigger another upgrade of RHOAM
4. Verify that the installPlan does not get approved until `status.stage` is `complete` and the `toVersion` field is empty.
