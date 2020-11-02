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

# C11B - Verify RHOAM install alert is firing

## Description

Verify that RHOAM operator alerts are in place and firing

More info: <https://issues.redhat.com/browse/INTLY-7395>

## Steps

1. Verify RHOAMOperatorInstallDelayed alert is present and firing:
2. Open OpenShift console in your browser
3. Login as admin
4. Go to Deployments, in the `redhat-managed-api-operator` namespace and click on `rhoam-operator`
5. In the details tab, decrease the pod count by clicking on the down arrow
   > RHOAM pod should be scaled to 0
6. In the left hand side menu, go to Monitoring >> Alerting and select only pending in the filter bar
   > RHOAMOperatorInstallDelayed should be in the list
7. Go to `rhoam-operator` deployment and scale the pod up again.
