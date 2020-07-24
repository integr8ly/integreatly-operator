---
environments:
  - osd-post-upgrade
targets:
  - 2.5.0
---

# C13 - Verify RHMI reconciliation alerts are firing

## Description

Verify that RHMI operator alerts are in place and firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHMIInstallationControllerIsNotReconciling and RHMIInstallationControllerStoppedReconciling alerts are present and firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Find route for Prometheus in `redhat-rhmi-middleware-monitoring-operator` namespace
5. Open its URL
6. Go to the Alerts tab and look for `RHMIInstallationControllerIsNotReconciling` or `RHMIInstallationControllerStoppedReconciling`
7. Click on the `expr`
8. In the query page change the **15m** to **1m** and click on the excute buttom `rhmi_status{stage="complete"} and on(namespace) rate(controller_runtime_reconcile_total{controller="installation-controller", result="success"}[1m]) == 0`
