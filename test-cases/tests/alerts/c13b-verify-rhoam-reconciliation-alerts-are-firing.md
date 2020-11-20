---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
---

# C13B - Verify RHOAM reconciliation alerts are firing

## Description

Verify that RHOAM operator alerts are in place and firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHOAMInstallationControllerIsNotReconciling and RHOAMInstallationControllerStoppedReconciling alerts are present and firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Find route for Prometheus in `redhat-rhoam-middleware-monitoring-operator` namespace
5. Open its URL
6. Go to the Alerts tab and look for `RHOAMInstallationControllerIsNotReconciling` or `RHOAMInstallationControllerStoppedReconciling`
7. Click on the `expr`
8. In the query page change the **15m** to **1m** and click on the execute buttom `rhoam_status{stage="complete"} and on(namespace) rate(controller_runtime_reconcile_total{controller="installation-controller", result="success"}[1m]) == 0`
