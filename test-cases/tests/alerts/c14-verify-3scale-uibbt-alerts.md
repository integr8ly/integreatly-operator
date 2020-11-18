---
estimate: 15m
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.6.0
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
---

# C14 - Verify 3scale UIBBT alerts

## Description

More info: <https://issues.redhat.com/browse/INTLY-9043>

## Steps

1. Scale down `system-app` in `redhat-rhmi-3scale`
   1. Login as kubeadmin
   2. **Workloads > Deployment Configs**
   3. Select `redhat-rhmi-3scale` project
   4. Scale `system-app` down to 0 pods
2. Check that all `ThreeScale**UIBBT` alerts are firing
   1. **Networking > Routes**
   2. Select `redhat-rhmi-middleware-monitoring-operator` project
   3. Open route for `alertmanager-route`
      > `ThreeScale**UIBBT` should be firing
