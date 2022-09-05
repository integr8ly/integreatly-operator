---
components:
  - monitoring
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.2.0
      - 1.6.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
      - 1.18.0
      - 1.21.0
      - 1.24.0
---

# E07B - Verify drill down links

## Description

Verify that drill down links in `resource usage by namespace` Grafana dashboard work correctly.

More info: <https://issues.redhat.com/browse/INTLY-5962>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhoam-observability` namespace
3. Open **Networking > Routes**
4. Click on **Grafana** route
5. Open `resource usage by namespace` dashboard
6. Click on drill down link for pods under **CPU quota** (verify at least 3 of them)
   > Dashboard with relevant data for that pod should be displayed
