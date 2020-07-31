---
components:
  - monitoring
environments:
  - osd-post-upgrade
targets:
  - 2.4.0
  - 2.8.0
---

# E08 - Verify values are correct in Resource usage dashboard

## Description

Verify that values (resource usage/requests/limits) in Grafana `Resource usage` dashboard are correct.

More info: <https://issues.redhat.com/browse/INTLY-7869>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
3. Open **Networking > Routes**
4. Click on **Grafana** route
5. Open `resource usage by pod` dashboard
6. Look at **memory** and **cpu** values for pods and compare them with actual cpu/memory usage/limits (in OpenShift console) (verify at least Keycloak and 2 other pods)
   > Vaues in the Grafana dashboard should correspond to actual values shown by OpenShift console
