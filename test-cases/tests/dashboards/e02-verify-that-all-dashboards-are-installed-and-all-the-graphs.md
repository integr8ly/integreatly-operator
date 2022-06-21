---
products:
  - name: rhoam
estimate: 15m
tags:
  - automated
---

# E02 - Verify that all dashboards are installed and all the graphs are filled with data

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/dashboards_data.go

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

[//]: # (TODO this is outlining the wrong namespace)
1. Open the RHMI Grafana Console in the `redhat-rhmi-middleware-monitoring-operator`
   > Verify that all **Dashboards** are present and all **Graphs** are active. It is acceptable for graphs to correctly report 0 activity.
