---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - osd-private-post-upgrade
estimate: 15m
tags:
  - per-release
---

# A29 - Verify Rate limit service pods are running

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed

## Steps

1. Go to `redhat-managed-api-marin3r` namespace
2. Verify there are 3 `ratelimit` pods running in the namespace (1 per AWS AZ)
3. Verify that the pods are running and there are no errors in the pods' log
