---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - osd-private-post-upgrade
    targets:
      - 1.0.0
estimate: 15m
tags:
  - automated
---

# A29 - Verify Rate limit service pods are running

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/alerts_firing.go

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed

## Steps

1. Go to `redhat-rhoam-marin3r` namespace
2. Verify there is 3 `ratelimit` pods running in the namespace
3. Verify that the pods are running and there are no errors in the pods' log
