---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.3.0
      - 2.7.0
---

# C10A - Verify additional BlackboxTargets

## Description

Verify that BlackboxTargets for Syndesis, User SSO, and Apicurito have been added to RHMI monitoring by checking Grafana detailed summary dashboard.

More info: <https://issues.redhat.com/browse/INTLY-6778>

## Steps

1. Open OpenShift console in your browser
2. Login as admin
3. Find route for Grafana in `redhat-rhmi-middleware-monitoring-operator` namespace
4. Open its URL
5. Open **Endpoints Detailed** dashboard
6. Verify that it contains status for all RHMI products including Syndesis, User SSO and Apicurito
