---
components:
  - monitoring
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.3.0
      - 1.4.0
      - 1.7.0
      - 1.10.0
      - 1.13.0
      - 1.16.0
      - 1.20.0
---

# E08B - Verify values are correct in Resource usage dashboard

## Description

Verify that values (resource usage/requests/limits) in Grafana `Resource usage` dashboard are correct.

More info: <https://issues.redhat.com/browse/INTLY-7869>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhoam-observability` namespace
3. Open **Networking > Routes**
4. Click on **Grafana** route
5. Open `resource usage by pod` dashboard
6. Look at **memory** and **cpu** values for pods and compare them with actual cpu/memory usage/limits (in OpenShift console) (verify at least Keycloak and 2 other pods)
   > Values in the Grafana dashboard should correspond to actual values shown by OpenShift console
   > Small differences are allowed - especially in memory consumption - due to slightly different promQL queries being used
