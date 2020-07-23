---
targets:
  - 2.5.0
---

# C11 - Verify RHMI install alert is firing

## Description

Verify that RHMI operator alerts are in place and firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHMIOperatorInstallDelayed alert is present and firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Go to Deployments, in the `redhat-rhmi-operator` namespace and click on `rhmi-operator`
5. In the details tab, decrease the pod count by clicking on the down arrow
   > RHMI pod should be scaled to 0
6. In the left hand side menu, go to Monitoring >> Alerting and select only pending in the filter bar
   > RHMIOperatorInstallDelayed should be in the list
7. Go to `rhmi-operator` deployment and scale the pod up again.
