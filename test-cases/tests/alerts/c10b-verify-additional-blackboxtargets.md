---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.2.0
      - 1.6.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
---

# C10B - Verify additional BlackboxTargets

## Description

Verify that BlackboxTargets for User SSO have been added to RHOAM monitoring by checking Grafana detailed summary dashboard.

## Steps

1. Open OpenShift console in your browser
2. Login as admin
3. Find route for Grafana in `redhat-rhoam-observability` namespace
4. Open its URL
5. Open **Endpoints Detailed** dashboard
6. Verify that it contains status for all RHOAM products including User SSO
