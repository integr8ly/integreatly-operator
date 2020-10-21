---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 15m
tags:
  - per-release
---

# E09 - Verify Grafana rate-limiting dashboard

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed
3. SSO IDP configured on a cluster

## Steps

1. Go to redhat-managed-api-customer-monitoring-operator namespace
2. Go to Routes -> grafana-route and login as customer-admin user using testing-idp
3. Verify the rate-limiting dashboard is present (TODO: more detailed step)
