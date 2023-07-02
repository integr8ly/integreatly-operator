# Setting Up OBO with RHOAM

This is a work-in-progress guide that details how to create an [OBO](https://github.com/rhobs/observability-operator) MonitoringStack CR along with the other required observability components. These components are created and reconciled via the [package-operator](https://package-operator.run/). Feel free to consult the [Epic Brief](https://docs.google.com/document/d/1HEk2D8V01n1tgGHLyhbA-OjlgP6POSJXasONwUoL60w/) and [Miro Board](https://miro.com/app/board/uXjVMcp5qQQ=/) for the OBO migration if more context is needed.

## Prerequisites
1. Provision an OSD cluster either through the [GUI](https://qaprodauth.console.redhat.com/openshift/create) or via the `ocm` [CLI](https://github.com/openshift-online/ocm-cli#installation)
2. `oc login` to the cluster from your local machine
3. Checkout the OBO [feature branch](https://github.com/integr8ly/integreatly-operator/tree/mgdapi-5727-obo)
4. Prepare the cluster for the RHOAM installation by running:
```bash
make cluster/prepare/local
```
**Note**: This command will also run `make cluster/prepare/rhoam-config` which will apply a [ClusterPackage CR](https://github.com/integr8ly/integreatly-operator/blob/mgdapi-5727-obo/config/hive-config/package.yaml) that by default uses this quay [config-image](https://quay.io/repository/integreatly/managed-api-service-config?tab=tags&tag=latest). That image contains all the components needed to set up an OBO observability stack - these components are declared in the [managed-tenants-bundles repository](https://gitlab.cee.redhat.com/ckyrillo/managed-tenants-bundles/-/tree/mgdapi-5727-obo/addons/rhoams/package/).
5. Install RHOAM to the cluster using a CatalogSource [installation](https://github.com/integr8ly/integreatly-operator/blob/master/docs/installation_guides/olm_installation.md)

## Access the Prometheus UI
1. OBO doesn't support the Prometheus UI, however it can still be accessed by setting up port forwarding on your local machine:
    ```bash
    oc port-forward services/prometheus-operated -n redhat-rhoam-operator-observability 9090:9090
    ```
2. The Prometheus UI can be reached by directing your browser to [http://127.0.0.1:9090/graph](http://127.0.0.1:9090/graph)

## Access the Alertmanager UI
1. OBO doesn't support the Alertmanager UI, however it can still be accessed by setting up port forwarding on your local machine:
    ```bash
    oc port-forward services/alertmanager-operated -n redhat-rhoam-operator-observability 9093:9093
    ```
2. The Alertmanager UI can be reached by directing your browser to [http://127.0.0.1:9093/#/status](http://127.0.0.1:9093/#/status)