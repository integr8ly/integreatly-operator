---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.3.0
      - 2.7.0
estimate: 15m
---

# M02A - Verify RHMI version metric is exposed

## Steps

Verify `rhmi_version` metric is present in Prometheus

1. Login into openshift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace, then under Networking >> Routes, click on the link for `prometheus` route
3. Login into Prometheus using Openshift by clicking on the button Log in with OpenShift
4. Look for `rhmi_version` metric in the Expression (press Shift+Enter for newlines) field and verify if metric is available
5. Select the metric and click in the Execute button
6. Check if the metric returns any data and if it has a version label with the current version of the operator `version="<RHMI_OPERATOR_VERSION>"`
   > rhmi_version{endpoint="http-metrics",instance="10.11.70.54:8383",job="rhmi-operator-metrics",namespace="redhat-rhmi-operator",pod="rhmi-operator-6c94bcf4ff-j99dl",service="rhmi-operator-metrics",stage="complete",version="2.3.0"}
