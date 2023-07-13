---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.5.0
      - 1.8.0
      - 1.11.0
      - 1.14.0
      - 1.18.0
      - 1.21.0
      - 1.24.0
      - 1.27.0
      - 1.30.0
      - 1.33.0
      - 1.36.0
estimate: 15m
tags:
  - destructive
---

# C14B - Verify 3scale UIBBT alerts

## Description

More info: <https://issues.redhat.com/browse/INTLY-9043>

## Steps

1. Login as kubeadmin
2. Scale down `3scale-operator` in `redhat-rhoam-3scale-operator`
   1. **Workloads > Deployment**
   2. Select `redhat-rhoam-3scale-operator` project
   3. Scale `3scale-operator` down to 0 pods
3. Scale down `system-app` in `redhat-rhoam-3scale`
   1. **Workloads > Deployment Configs**
   2. Select `redhat-rhoam-3scale` project
   3. Scale `system-app` down to 0 pods
4. Check that all `ThreeScale**UIBBT` alerts are firing
   1. **Networking > Routes**
   2. Select `redhat-rhoam-operator-operator-observability` project
   3. Open route for `alertmanager`
      > `ThreeScale**UIBBT` should be firing
5. Scale 3scale operator back up
   1. **Workloads > Deployment**
   2. Select `redhat-rhoam-3scale-operator` project
   3. Scale `3scale-operator` up to 1 pods
