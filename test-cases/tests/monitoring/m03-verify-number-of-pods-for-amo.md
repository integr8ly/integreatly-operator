---
targets:
  - 2.5.0
---

# M03 - Verify number of pods for AMO

There was a bug in application-monitoring-operator - after RHMI upgrade there were 2 pods for it, when there should be just one.

More info: <https://issues.redhat.com/browse/INTLY-8046>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
3. Open **Workloads > Pods**
   > There should be only one pod for `application-monitoring-operator`
