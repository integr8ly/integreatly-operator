---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.13.0
estimate: 1h
tags:
  - manual-selection
---

# A41 - Verify migration to Observability Operator was successful

## Steps

1. Verify that there are two namespaces `redhat-rhoam-observability-operator` and `redhat-rhoam-observabilty`
2. Verify that the Grafana, Prometheus, Alert Manager in redhat-rhoam-observability are all running
3. Verify that only the observability-operator pod is running in the `redhat-rhoam-observability-operator` namespace
4. Verify a new block of the RHMI CR status is present

```
observability:
      name: observability
      phase: completed
      products:
        observability:
          host: ''
          name: observability
          operator: 3.0.8
          status: completed
          version: 3.0.8
```

5. Verify that the config-map redhat-rhoam-installation-config in redhat-rhoam-operator namespace contains an observability block with the following

```
NAMESPACE: redhat-rhoam-observability
OPERATOR_NAMESPACE: redhat-rhoam-observability-operator
```

6. Verify alert manager route with name alertmanager is present

```
oc get routes alertmanager -n redhat-rhoam-observability
```

7. Verify you can login to alertmanager using the route
8. Verify prometheus route with name prometheus is present

```
oc get routes prometheus -n redhat-rhoam-observability
```

9. Verify you can login to prometheus using the route

10. Verify Prometheus version installed is v2.29.2 under Status -> Runtime Information & Build Information

11. Verify alert manager version installed in v0.22.2 under Status -> Version Information

12. Verify resource specifications

```
oc get alertmanager alertmanager -n redhat-rhoam-observability -o json| jq '.spec.resources'
oc get grafana grafana -n redhat-rhoam-observability -o json| jq '.spec.resources'
oc get deployment grafana-operator -n redhat-rhoam-observability -o json| jq '.spec.template.spec.containers[0].resources'
oc get prometheus prometheus -n redhat-rhoam-observability -o json| jq '.spec.resources'
oc get deployment prometheus-operator -n redhat-rhoam-observability -o json| jq '.spec.template.spec.containers[0].resources'

# {
#   "requests": {
#     "memory": "200Mi"
#   }
# }
# {
#   "limits": {
#     "cpu": "500m",
#     "memory": "1Gi"
#   },
#   "requests": {
#     "cpu": "100m",
#     "memory": "256Mi"
#   }
# }
# {}
# {
#   "requests": {
#     "memory": "400Mi"
#   }
# }
# {
#   "limits": {
#     "cpu": "200m",
#     "memory": "400Mi"
#   },
#   "requests": {
#     "cpu": "100m",
#     "memory": "200Mi"
#   }
# }
```

13. Verify retention and storage is specified in Prometheus CR

```
oc get prometheus prometheus -n redhat-rhoam-observability -o json | jq '.spec.storage'
oc get prometheus prometheus -n redhat-rhoam-observability -o json | jq '.spec.retention'
```

14. Verify PVC is created for prometheus

```
oc get pvc -n redhat-rhoam-observability
```

_Monitoring uninstall_

1. Two of the middleware-monitoring namespaces should be NOT present excluding redhat-rhoam-monitoring which remains with some configuration.
2. Navigate to openshift-monitoring -query rhoam_version check for instances of the metric with the label to_version 1.13.0, check on the graph how long this instance of the metric was in place. I have logged 7 minutes for one installation. We need to keep the upgrade under 10 minutes.

_Customer grafana_

1. Navigate to the customer grafana, there should be metrics present on the customer dashboard.
   You can also check this by checking the grafanadatasource in the project. the url should be 'http://prometheus.redhat-rhoam-observability.svc:9090' This will need to be updated when the route is updated in another JIRA.

_Prometheus_

1. Navigate to Prometheus and verify that the DeadMansSwitch Alert is present.
2. Verify that the probes can be seen on the Targets page in the redhat-rhoam-observability Prometheus instance and that when expanded, each target lists its State as UP

_Federation_

1. Verify that the observability contains the following block

```
federatedMetrics:
      - >-
        'kubelet_volume_stats_used_bytes{endpoint="https-metrics",namespace=~"redhat-rhoam-.*"}'
      - >-
        'kubelet_volume_stats_available_bytes{endpoint="https-metrics",namespace=~"redhat-rhoam-.*"}'
      - >-
        'kubelet_volume_stats_capacity_bytes{endpoint="https-metrics",namespace=~"redhat-rhoam-.*"}'
      - >-
        'haproxy_backend_http_responses_total{route=~"^keycloak.*",
        exported_namespace=~"redhat-rhoam-.*sso$"}'
      - '''{ service="kube-state-metrics" }'''
      - '''{ __name__=~"node_namespace_pod_container:.*" }'''
      - '''{ __name__=~"instance:.*" }'''
      - '''{ __name__=~"container_memory_.*" }'''
      - '''{ __name__=~":node_memory_.*" }'''
      - '''{ __name__=~"csv_.*" }'''
```

2. Verify that the secret additional-scrape-configs (in OpenShift console go to `redhat-rhoam-observability` namespace -> secrets -> `additional-scrape-configs` contains the following block

```
match[]: ['kubelet_volume_stats_used_bytes{endpoint="https-metrics",namespace=~"redhat-rhoam-.*"}','kubelet_volume_stats_available_bytes{endpoint="https-metrics",namespace=~"redhat-rhoam-.*"}','kubelet_volume_stats_capacity_bytes{endpoint="https-metrics",namespace=~"redhat-rhoam-.*"}','haproxy_backend_http_responses_total{route=~"^keycloak.*", exported_namespace=~"redhat-rhoam-.*sso$"}','{ service="kube-state-metrics" }','{ __name__=~"node_namespace_pod_container:.*" }','{ __name__=~"instance:.*" }','{ __name__=~"container_memory_.*" }','{ __name__=~":node_memory_.*" }','{ __name__=~"csv_.*" }']
```

3. Verify that metrics are present in promethues by navigating to prometheus and running some queries.

_Grafana configuration_

1. Log into the Grafana found in the redhat-rhoam-observability namespace. Get route

```
oc get routes -n redhat-rhoam-observability --no-headers | grep grafana | awk '{print $2}' | xargs -I {} echo https://{}
```

2. Check that the dashboards are all in grafana.

current problem is the sso dashboards are not being picked up
Some Data (cro/sso/3scale) will not be showing in the dashboards. This is should be fixed in a different ticket

Expected Dashboards:

- Critical SLO summary
- CRO Resources
- Endpoints Detailed
- Endpoints Report
- Endpoints Summary
- Resource Usage By Namespace
- Resource Usage for Cluster
- SLO SSO Availability - 5 xx HAProxy Errors
- Keycloak Metrics (Folders: redhat-rhoam-rhsso, redhat-rhoam-user-sso)
