---
products:
  - name: rhoam
tags:
  - automated
---

# M04 - Verify rhmi_status metric

More info: <https://issues.redhat.com/browse/INTLY-8120>

**Automated Test**: [rhmicr_metrics.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/rhmicr_metrics.go)

## Steps

[//]: # (TODO this is using the wrong namespace)
1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
3. Open **Networking > Routes**
4. Click on **Prometheus** route
5. Type `rhmi_status` into **Expression** field and click **Execute**
   > Should return 1 metric with the stage label having a value of the current stage (e.g. `rhmi_status{stage="bootstrap"}` - `1`)
