---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.13.0
tags:
  - manual-selection
---

# E10 - Validate updated Grafana Dashboards and Alerts

## Prerequisites

1. RHOAM 1.12.0 installed on a cluster with OSD version < 4.7.21
2. RHOAM 1.13.0 installed on a cluster with OSD version >= 4.8.x
3. Login to OpenShift console in both clusters as a user with **cluster-admin** role (kubeadmin):

## Steps

1. In each of the two clusters, go to grafana route (for RHOAM 1.12.0: redhat-rhoam-middleware-monitoring-operator namespace, for RHOAM 1.13.0: redhat-rhoam-observability) and log in
2. Click on Home and find the following dashboards:

- Resource Usage for Cluster
- Resource Usage By Namespace
- Resource Usage By Pod

> Note if there are any differences in graphs between those 2 clusters.
> Each Dashboard should have full metrics expected, e.g. in each memory and cpu table there should appear 5 columns (actual, requested, %actual against requested, limits, %actual against limits) - note deploy pods in 3scale namespace wont return any data, please select another pod.
> There should be no single single stat or graphs with no data or N/A or error saying Only queries that return single series/table is supported

3. On OSD version >= 4.8 cluster, open Prometheus route (in redhat-rhoam-observability namespace) and log in
4. `ClusterSchedulableMemoryLow` - find the alert in prometheus. Click on the query to open it up in the query page and remove \* 85 from the end of the query.
   > Make sure that some data is getting returned.
5. `ClusterSchedulableCPULow` - find the alert in prometheus. Click on the query to open it up in the query page and remove \* 85 from the end of the query.
   > Make sure that some data is getting returned.
6. `MultiAZPodDistribution` - remove rhoam_version{to_version=""} and remove == 1 from the end of the expression
   > Make sure that now data is returned
7. `ThreeScaleContainerHighMemory` - find the alert in prometheus. Click on the query to open it up in the query page and remove > 90.
   > Make sure some data is returned - Note this alert might be firing but this is to be addressed in -> https://issues.redhat.com/browse/MGDAPI-2598 > `ThreeScaleContainerHighCPU` - find the alert in prometheus. Click on the query to open it up in the query page and remove > 90.
   > Also make sure some data is returned
