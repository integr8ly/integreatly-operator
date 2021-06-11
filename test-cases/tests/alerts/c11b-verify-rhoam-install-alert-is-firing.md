---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.4.0
tags:
  - destructive
  - manual-selection
---

# C11B - Verify RHOAM install alert is firing

## Description

> Obsolete, the alert in question should not fire if RHOAM operator is down, it is just for installations/upgrades.

Verify that RHOAM operator alerts are in place and working as expected

## Steps

1. Verify RHOAMOperatorInstallDelayed alert is present and firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Go to Deployments, in the `redhat-rhoam-operator` namespace and click on `rhmi-operator`
5. In the details tab, decrease the pod count by clicking on the down arrow
   > rhmi-operator pod should be scaled to 0
6. In the left hand side menu, go to Monitoring >> Alerting and select only pending in the filter bar
   > RHOAMOperatorInstallDelayed should be in the list
7. Go to `rhmi-operator` deployment and scale the pod up again.
