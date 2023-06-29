# Setting Up OBO with RHOAM

This is a work-in-progress guide that details how to create an [OBO](https://github.com/rhobs/observability-operator) MonitoringStack CR with basic configuration alongside RHOAM. Note: this doc is not meant to walkthrough setting up a production-worthy monitoring solution that includes all the features that RHOAM's current monitoring solution ([OO](https://github.com/redhat-developer/observability-operator)) employs - this guide simply provides a barebones Prometheus and Alertmanager with minimal configuration using OBO.

Feel free to consult the [Epic Brief](https://docs.google.com/document/d/1HEk2D8V01n1tgGHLyhbA-OjlgP6POSJXasONwUoL60w/) and [Miro Board](https://miro.com/app/board/uXjVMcp5qQQ=/) for the OBO migration if more context is needed.

## Prerequisites
1. Provision an OSD cluster either through the [GUI](https://qaprodauth.console.redhat.com/openshift/create) or via the `ocm` [CLI](https://github.com/openshift-online/ocm-cli#installation)
2. `oc login` to the cluster from your local machine
3. Deploy RHOAM to the cluster using a CatalogSource [installation](https://github.com/integr8ly/integreatly-operator/blob/master/docs/installation_guides/olm_installation.md)

## Create the MonitoringStack CR
1. Apply this YAML [file](./obo_crs.yaml) to the cluster to create the MonitoringStack CR, `alertmanager-rhoam` Secret, `openshift-monitoring-federation` ServiceMonitor, and the required RBAC:
    ```bash
    oc apply -f docs/observability/obo_crs.yaml
    ```

## Access the Prometheus UI
1. OBO currently doesn't support the Prometheus UI, however it can still be accessed by setting up port forwarding on your local machine:
    ```bash
    oc port-forward services/prometheus-operated -n redhat-rhoam-operator 9090:9090
    ```
2. The Prometheus UI can be reached by directing your browser to [http://127.0.0.1:9090/graph](http://127.0.0.1:9090/graph)

## Access the Alertmanager UI
1. OBO currently doesn't support the Alertmanager UI, however it can still be accessed by setting up port forwarding on your local machine:
    ```bash
    oc port-forward services/alertmanager-operated -n redhat-rhoam-operator 9093:9093
    ```
2. The Alertmanager UI can be reached by directing your browser to [http://127.0.0.1:9093/#/status](http://127.0.0.1:9093/#/status)