---
targets:
  - 2.5.0
---

# C11 - Verify RHMI installation alerts are firing

## Description

Verify that RHMI operator alerts are in place and firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHMIOperatorInstallCompleted alert is present and firing:
   1. Open OpenShift console in your browser and login as admin
   2. Login as admin
   3. Go to Deployments, in the `redhat-rhmi-operator` namespace and click on `rhmi-operator`
   4. In the details tab, decrease the pod count by clicking on the down arrow
      > RHMI pod should be scaled to 0
   5. In the left hand side menu, go to Monitoring >> Alerting and select only Firing in the filter bar
      > RHMIInstallationCompleted should be in the list
   6. Go to `rhmi-operator` deployment and scale the pod up again.
2. Verify RHMIInstallationControllerIsNotReconciling and RHMIInstallationControllerStoppedReconciling alerts are present and firing:
   1. Open OpenShift console in your browser and login as admin
   2. Login as admin
   3. Find route for Prometheus in `redhat-rhmi-middleware-monitoring-operator` namespace
   4. Open its URL
   5. Go to the Alerts tab and look for `RHMIInstallationControllerIsNotReconciling` or `RHMIInstallationControllerStoppedReconciling`
   6. Click on the `expr`
   7. In the query page change the **15m** to **1m** and click on the excute buttom `rhmi_status{stage="complete"} and on(namespace) rate(controller_runtime_reconcile_total{controller="installation-controller"}[1m]) == 0`
      > It should return the `rhmi_status` metric
