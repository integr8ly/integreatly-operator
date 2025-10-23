[![codecov](https://codecov.io/gh/integr8ly/integreatly-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/integr8ly/integreatly-operator)
# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling managed products.

An Integreatly Operator can be installed using three different flavours: `managed-api` or `multitenant-managed-api`

To switch between the three you can use export the `INSTALLATION_TYPE` env or use it in conjunction with any of the make commands referenced in this README

### Installed products

The operator installs the following products:

### managed-api

- 3scale
- RHSSO (both a cluster instance, and a user instance)
- Marin3r

### multitenant-managed-api

- 3scale
- RHSSO (cluster instance)
- Marin3r

## Prerequisites

- [operator-sdk](https://github.com/operator-framework/operator-sdk) version v1.39.0.
- [go](https://golang.org/dl/) version 1.23+
- [moq](https://github.com/matryer/moq)
- [oc](https://docs.okd.io/latest/cli_reference/openshift_cli/getting-started-cli.html) version v4.6+
- [yq](https://github.com/mikefarah/yq) version v4.30+
- [jq](https://github.com/stedolan/jq)
- [promtool](https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules)
- Access to an Openshift v4.6.0+ cluster
- A user with administrative privileges in the OpenShift cluster

After installation, the following commands must be run to avoid a known issue related to the Moq package:
```shell
make code/compile
go install github.com/matryer/moq
```

## Local Development
Ensure that the cluster satisfies minimal requirements:
- RHOAM (managed-api and multitenant-managed-api): 18 vCPU. More details can be found in the [service definition](https://access.redhat.com/articles/5534341) 
  under the "Resource Requirements" section.



### 1. Clone the integreatly-operator
Only if you haven't already cloned. Otherwise, navigate to an existing copy. 
```sh
mkdir -p $GOPATH/src/github.com/integr8ly
cd $GOPATH/src/github.com/integr8ly
git clone https://github.com/integr8ly/integreatly-operator
cd integreatly-operator
```

### 2. Cluster size guidelines

For development work the required vcpu and ram can be lower than that stated in the [service definition](https://access.redhat.com/articles/5534341#scalability-and-service-levels-15).
Different quotas require different values.
Table belong are typical requested values needed for RHOAM on a cluster with cluster storage set to True.

| Quota        | vCPU     | RAM     |
|--------------|----------|---------|
| 100 Thousand | 6.5 vCPU | 22 Gb   |
| 1 Million    | 6.5 vCPU | 22 Gb   |
| 5 Million    | 8 vCPU   | 24 Gb   |
| 10 Million   | 8.5 vCPU | 24 Gb   |
| 20 Million   | 9.5 vCPU | 24 Gb   |
| 50 Million   | 14 vCPU  | 26.5 Gb |

### 3. Prepare your cluster

If you are working against a fresh cluster it will need to be prepared using the following. 
Ensure you are logged into a cluster by `oc whoami`.
Include the `INSTALLATION_TYPE`. See [here](#4-configuration-optional) about this and other optional configuration variables.
```shell
INSTALLATION_TYPE=<managed-api/multitenant-managed-api> make cluster/prepare/local
```


### 4. Configuration (optional)

If you are running RHOAM against a cluster which is smaller than the requirements listed above, you 
should use the IN_PROW variable, otherwise the installation will not complete. 
If you have a cluster which meets the requirements, this step can be skipped.
Please see the table below for other configuration options.

```shell script
INSTALLATION_TYPE=managed-api IN_PROW=true USE_CLUSTER_STORAGE=<true/false> make deploy/integreatly-rhmi-cr.yml
```

| Variable            | Options                                    |     Type     | Default       | Details                                                                                              |
|---------------------|--------------------------------------------|:------------:|---------------|------------------------------------------------------------------------------------------------------|
| INSTALLATION_TYPE   | `managed-api` or `multitenant-managed-api` | **Required** | `managed-api` | Manages installation type. `managed-api` for RHOAM. `multitenant-managed-api` for Multitenant RHOAM. |
| IN_PROW             | `true` or `false`                          |   Optional   | `false`       | If `true`, reduces the number of pods created. Use for small clusters                                |
| USE_CLUSTER_STORAGE | `true` or `false`                          |   Optional   | `true`        | If `true`, installs application to the cloud provider. Otherwise installs to the OpenShift.          |


### 5. Run integreatly-operator
Include the `INSTALLATION_TYPE` if you haven't already exported it. 
The operator can now be run locally:
```shell
INSTALLATION_TYPE=<managed-api/multitenant-managed-api> make code/run
```
If you want to run the operator from a specific image, you can specify the image and run `make cluster/deploy`
```shell
IMAGE_FORMAT=<image-registry-address> INSTALLATION_TYPE=managed-api  make cluster/deploy
```

*Note:* if the operator doesn't find an RHMI cr, it will create one (Name: `rhoam`).

| Variable            | Options   |   Type   | Default                        | Details                                                                                                                                    |
|---------------------|-----------|:--------:|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------|
| PRODUCT_DECLARATION | File path | Optional | `./products/installation.yaml` | Specifies how RHOAM install the product operators, either from a local manifest, an index, or an included bundle. Only applicable to RHOAM |

### 6. Validate installation 

Use following commands to validate that installation succeeded:

For `RHOAM` (managed-api): `oc get rhmi rhoam -n redhat-rhoam-operator -o jsonpath='{.status.stage}{"\n"}'`

For `RHOAM Multitenant` (multitenant-managed-api): `oc get rhmi rhoam -n sandbox-rhoam-operator -o jsonpath='{.status.stage}{"\n"}'`

Once the installation completed the command wil result in following output:  
```yaml
"complete"
```

## Uninstalling RHOAM
### Local and OLM installation type
If you installed RHOAM locally or via OLM then you can uninstall one of two ways:

- for local installation use the namespace that the RHMI CR was created in.
- for OLM installation use the `redhat-rhoam-operator Namespace`


A) Create a configmap and add a deletion label (Preferred way of uninstallation).
```sh 
oc create configmap managed-api-service -n <NAMESPACE>
oc label configmap managed-api-service api.openshift.com/addon-managed-api-service-delete=true -n <NAMESPACE>
```

B) Delete the RHOAM cr.
```sh 
oc delete rhmi rhoam -n <NAMESPACE>
```

In both scenarios wait until the RHOAM cr is removed and then run the following command to delete the namespace.
```sh 
oc delete namespace <NAMESPACE>
```

#### Note: After uninstalling RHOAM you should clean up the cluster by running the following command.

```sh
make cluster/cleanup && make cluster/cleanup/crds
```


## More Info
More info can be found in the docs folder and at the [Integreatly Read the Docs site](https://integreatly-operator.readthedocs.io/en/latest/).