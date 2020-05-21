---
estimate: 15m
---

# E02 - Verify that all dashboards are installed and all the graphs are filled with data

## Description

Only RHMI provided dashboards are required to be verified.

**Dashboards:**

- Endpoints
  - Endpoints Detailed
  - Endpoints Report
  - Endpoints Summary
- Resource Usage
  - Resource Usage By Namespace
  - Resource Usage By Pod
  - Resource Usage for Cluster

## Steps

1. Open the RHMI Grafana Console in the `redhat-rhmi-middleware-monitoring-operator`
   > Verify that all **Dashboards** are present and all **Graphs** are active. It is acceptable for graphs to correctly report 0 activity.
