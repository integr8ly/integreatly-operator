---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
tags:
  - automated
---

# E09 - Verify Customer Grafana Route is accessible

**Automated Test**: [grafana_routes.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/grafana_routes.go)

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed
3. SSO IDP configured on a cluster

## Steps

1. Go to redhat-rhmi-customer-monitoring namespace
2. Go to Routes -> grafana-route and login as customer-admin user using testing-idp
   > Verify that you can successfully log in to the Grafana
