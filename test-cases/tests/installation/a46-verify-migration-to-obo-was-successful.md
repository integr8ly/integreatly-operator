# A46 - Verify migration to OBO was successful

## Steps

1. Verify that there is one namespace `redhat-rhoam-operator-observability`
2. Verify that the Prometheus pod, Alert Manager pods (2) and the blackbox-exporter pod in redhat-rhoam-operator-observability are all running
3. Verify alertmanager config Secret exists

```shell script
oc describe secret alertmanager-rhoam -n redhat-rhoam-operator-observability
```

4. Verify black-box-config ConfigMap exists

```shell script
oc describe configmap black-box-config -n redhat-rhoam-operator-observability
```

5. Verify retention and storage is specified in Prometheus CR

```shell script
oc get prometheus.monitoring.rhobs rhoam -n redhat-rhoam-operator-observability -o json | jq -r '.spec.retention, .spec.storage'
```

6. Verify PVC is created for prometheus

```shell script
oc get pvc -n redhat-rhoam-operator-observability
```

_Prometheus_

7. Verify that the DeadMansSwitch Alert is present.

```shell script
oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate http://localhost:9090/api/v1/alerts | jq -r '.data.alerts'
```

8. Click On Dashboards -> Manage

_Federation_

9. Verify that the observability contains the following block

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
      - '''{ service="node-exporter" }'''
      - '''{ __name__=~"node_namespace_pod_container:.*" }'''
      - '''{ __name__=~"node:.*" }'''
      - '''{ __name__=~"instance:.*" }'''
      - '''{ __name__=~"container_memory_.*" }'''
      - '''{ __name__=~":node_memory_.*" }'''
      - '''{ __name__=~"csv_.*" }'''
```

- Test out the federated metrics

```shell script
oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate "http://localhost:9090/api/v1/query?query=kubelet_volume_stats_used_bytes{endpoint=\"https-metrics\",namespace=~\"redhat-rhoam-.*\"}" | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate "http://localhost:9090/api/v1/query?query=kubelet_volume_stats_available_bytes{endpoint=\"https-metrics\",namespace=~\"redhat-rhoam-.*\"}" | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate "http://localhost:9090/api/v1/query?query=kubelet_volume_stats_capacity_bytes{endpoint=\"https-metrics\",namespace=~\"redhat-rhoam-.*\"}" | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query=haproxy_backend_http_responses_total{route=~"^keycloak.*",exported_namespace=~"redhat-rhoam-.*sso$"}' | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={service="kube-state-metrics"}' | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={service="node-exporter"}' | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={__name__=~"node_namespace_pod_container:.*"}' | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={__name__=~"node:.*"}' | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={__name__=~"instance:.*"}' | jq -r

# **Note** Result generated too large and unable to be formatted
oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={__name__=~"container_memory_.*"}' >> result.json

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={__name__=~":node_memory_.*"}' | jq -r

oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9090/api/v1/query?query={__name__=~"csv_.*"}' | jq -r

```

- Accessing the Prometheus UI

```shell script
oc port-forward -n redhat-rhoam-operator-observability prometheus-rhoam-0 9090:9090
```

Follow the link and log in using cluster credentials if needed.
