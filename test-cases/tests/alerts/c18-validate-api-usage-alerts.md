---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 1h
tags:
  - per-release
---

# C18 - Validate API usage alerts

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed

## Steps

1. Validate that rate-limiting alerts ConfigMap has been created

```
oc get cm rate-limit-alerts -n redhat-managed-api-operator
```

2. Verify that level1, level2 and level3 rate-limiting alerts are present

```
oc get prometheusrules -n redhat-managed-api-marin3r
```

3. (TODO: modify the alerts configmap - configmap name?)
4. Verify that the configmap is being reconciled and PrometheusRules CRs are updated with the changes in the configmap
