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

# C13B - Verify RHOAM reconciliation alerts are not firing

## Description

Verify that RHOAM operator alerts are in place and are not firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHOAMnstallationControllerIsInReconcilingErrorState alerts are present and are not firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Find route for Prometheus in `redhat-rhoam-middleware-monitoring-operator` namespace
5. Open its URL
6. Go to the Alerts tab and look for `RHOAMnstallationControllerIsInReconcilingErrorState`
7. Click on the `expr`
8. In the query page click on the execute button `rhmi_status{stage="complete"} and on(namespace) rate(controller_runtime_reconcile_total{controller="installation-controller", result="error"}[1m]) >= 1` should return no data
