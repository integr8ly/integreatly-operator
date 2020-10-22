---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 15m
tags:
  - per-release
---

# E09 - Verify customer dashboards exist and shows a correct data

**Automated Test**: [dashboards_exist.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/dashboards_exist.go)

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed
3. SSO IDP configured on a cluster

## Steps

1. Run the following command and log in as a `customer-admin` user using testing-idp

```
open "https://$(oc get routes grafana-route -n redhat-rhmi-customer-monitoring-operator -o jsonpath='{.spec.host}')"
```

2. Verify the "3Scale Api Rate Limiting" dashboard is present and shows some data
3. TODO (Some work around the dashboard is still WIP. This test case will be updated)
