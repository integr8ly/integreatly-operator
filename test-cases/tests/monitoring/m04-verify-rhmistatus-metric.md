---
tags:
  - 2.4.0
---

# M04 - Verify rhmi_status metric

More info: <https://issues.redhat.com/browse/INTLY-8120>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
3. Open **Networking > Routes**
4. Click on **Prometheus** route
5. Type `rhmi_status` into **Expression** field and click **Execute**
   > Should return 8 metrics, one of them should be of value `1` (indicating current status) and the rest `0`
