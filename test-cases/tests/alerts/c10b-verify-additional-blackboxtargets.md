---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
---

# C10b - Verify additional BlackboxTargets

## Description

Verify that BlackboxTargets for User SSO have been added to RHOAM monitoring by checking Grafana detailed summary dashboard.

More info: <https://issues.redhat.com/browse/INTLY-6778>

## Steps

1. Open OpenShift console in your browser
2. Login as admin
3. Find route for Grafana in `redhat-managed-api-middleware-monitoring-operator` namespace
4. Open its URL
5. Open **Endpoints Detailed** dashboard
6. Verify that it contains status for all RHOAM products including User SSO
