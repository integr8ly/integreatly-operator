---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.5.0
---

# C13A - Verify RHMI reconciliation alerts are not firing

## Description

Verify that RHMI operator alerts are in place not firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHMIInstallationControllerIsInReconcilingErrorState alerts are present and not firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Find route for Prometheus in `redhat-rhmi-middleware-monitoring-operator` namespace
5. Open its URL
6. Go to the Alerts tab and look for `RHMIInstallationControllerIsInReconcilingErrorState`
7. Click on the `expr`
8. In the query page click on the execute button `rhmi_status{stage="complete"} and on(namespace) rate(controller_runtime_reconcile_total{controller="installation-controller", result="error"}[1m]) >= 1` should return no data
