---
components:
  - monitoring
environments:
  - osd-post-upgrade
targets:
  - 2.4.0
  - 2.7.0
---

# E07 - Verify drill down links

## Description

Verify that drill down links in `resource usage by namespace` Grafana dashboard work correctly.

More info: <https://issues.redhat.com/browse/INTLY-5962>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
3. Open **Networking > Routes**
4. Click on **Grafana** route
5. Open `resource usage by namespace` dashboard
6. Click on drill down link for pods under **CPU quota** (verify at least 3 of them)
   > Dashboard with relevant data for that pod should be displayed
