# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling managed products.

An Integreatly Operator can be installed using three different flavours: `managed`, `managed-api` or `multitenant-managed-api`

To switch between the three you can use export the `INSTALLATION_TYPE` env or use it in conjunction with any of the make commands referenced in this README

### Installed products

The operator installs the following products:

### managed

- AMQ Online
- AMQ Streams
- Codeready
- Fuse
- Nexus
- RHSSO (both a cluster instance, and a user instance)
- 3scale
- Integreatly solution explorer

### managed-api

- 3scale
- RHSSO (both a cluster instance, and a user instance)
- Marin3r

### multitenant-managed-api

- 3scale
- RHSSO (cluster instance)
- Marin3r

## Prerequisites

- [operator-sdk](https://github.com/operator-framework/operator-sdk) version v1.12.0.
- [go](https://golang.org/dl/) version 1.16.7+
- [moq](https://github.com/matryer/moq)
- [oc](https://docs.okd.io/latest/cli_reference/openshift_cli/getting-started-cli.html) version v4.6+
- [yq](https://github.com/mikefarah/yq) version v4+
- [jq](https://github.com/stedolan/jq)   
- Access to an Openshift v4.6.0+ cluster
- A user with administrative privileges in the OpenShift cluster

After installation, the following commands must be run to avoid a known issue related to the Moq package:
```shell
make code/compile
go install github.com/matryer/moq
```

## Using `ocm` for installation of RHMI

If you want to test your changes on a cluster, the easiest solution would be to spin up OSD 4 cluster using `ocm`.
See [here](https://github.com/integr8ly/delorean/tree/master/docs/ocm) for an up to date guide on how to do this.


## Local Development
Ensure that the cluster satisfies minimal requirements: 
- RHMI (managed): 26 vCPU 
- RHOAM (managed-api and multitenant-managed-api): 18 vCPU. More details can be found in the [service definition](https://access.redhat.com/articles/5534341) 
  under the "Resource Requirements" section

### 1. Clone the integreatly-operator
Only if you haven't already cloned. Otherwise, navigate to an existing copy. 
```sh
mkdir -p $GOPATH/src/github.com/integr8ly
cd $GOPATH/src/github.com/integr8ly
git clone https://github.com/integr8ly/integreatly-operator
cd integreatly-operator
```

### 2. Prepare your cluster

If you are working against a fresh cluster it will need to be prepared using the following. 
Ensure you are logged into a cluster by `oc whoami`.
Include the `INSTALLATION_TYPE`. See [here](#3-configuration-optional) about this and other optional configuration variables.
```shell
INSTALLATION_TYPE=<managed/managed-api> make cluster/prepare/local
```


### 3. Configuration (optional)

If you are running RHOAM against a cluster which is smaller than the requirements listed above, you 
should use the IN_PROW variable, otherwise the installation will not complete. 
If you have a cluster which meets the requirements, this step can be skipped.
Please see the table below for other configuration options.

```shell script
INSTALLATION_TYPE=managed-api IN_PROW=true USE_CLUSTER_STORAGE=<true/false> make deploy/integreatly-rhmi-cr.yml
```

| Variable | Options | Type | Default | Details |
|----------|---------|:----:|---------|-------|
| INSTALLATION_TYPE     | `managed`, `managed-api` or `multitenant-managed-api` | **Required** |`managed`  | Manages installation type. `managed` stands for RHMI. `managed-api` for RHOAM. `multitenant-managed-api` for Multitenant RHOAM. |
| IN_PROW               | `true` or `false`         | Optional      |`false`    | If `true`, reduces the number of pods created. Use for small clusters |
| USE_CLUSTER_STORAGE   | `true` or `false`         | Optional      |`true`     | If `true`, installs application to the cloud provider. Otherwise installs to the OpenShift. |


### 4. Run integreatly-operator
Include the `INSTALLATION_TYPE` if you haven't already exported it. 
The operator can now be run locally:
```shell
INSTALLATION_TYPE=<managed/managed-api/multitenant-managed-api> make code/run
```
If you want to run the operator from a specific image, you can specify the image and run `make cluster/deploy`
```shell
IMAGE_FORMAT=<image-registry-address> INSTALLATION_TYPE=managed-api  make cluster/deploy
```

*Note:* if the operator doesn't find an RHMI cr, it will create one (Name: `rhmi/rhoam`).

| Variable | Options | Type | Default | Details |
|----------|---------|:----:|---------|-------|
| PRODUCT_DECLARATION | File path | Optional |`./products/installation.yaml` | Specifies how RHOAM install the product operators, either from a local manifest, an index, or an included bundle. Only applicable to RHOAM |

### 5. Validate installation 

Use following commands to validate that installation succeeded:

For `RHMI` (managed): `oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq .status.stage`

For `RHOAM` (managed-api): `oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage `

For `RHOAM Multitenant` (multitenant-managed-api): `oc get rhmi rhoam -n sandbox-rhoam-operator -o json | jq .status.stage `

Once the installation completed the command wil result in following output:  
```yaml
"complete"
```

## Deploying to a Cluster with OLM and the Bundle Format

In order to create a bundle and/or deploy RHMI or RHOAM with OLM follow refer to [this](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/) document.

For more details refer to [this](https://github.com/RHCloudServices/integreatly-help/blob/master/guides/olm/installing-rhmi-bundle-format.md#installing-rhmi-through-olm-with-bundle-format) readme file.

## 	Identity Provider setup
### Set up testing IDP for OSD cluster
You can use the `scripts/setup-sso-idp.sh` script to setup a "testing-idp" realm in a cluster SSO instance and add it as IDP of your OSD cluster.
With this script you will get few regular users - test-user[01-10] and few users that will be added to dedicated-admins group - customer-admin[01-03].

Prerequisites:
- `oc` command available on your machine (the latest version can be downloaded [here](https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/))
- `ocm` command available ( the newest CLI can be downloaded [here](https://github.com/openshift-online/ocm-cli/releases) and you install it with `mv (your downloaded file) /usr/local/bin/ocm`) (necessary only if using OSD cluster)
- OC session with cluster admin permissions in a target cluster
- OCM session (necessary only if using OSD cluster)

| Variable                  | Format  | Type     | Default        | Details                                                                     |
|---------------------------|---------|:--------:|----------------|-----------------------------------------------------------------------------|
| PASSWORD                  | string  | Optional | _None_         | If empty, a random password is generated for the testing users.             |
| DEDICATED_ADMIN_PASSWORD  | string  | Optional | _None_         | If empty, a random password is generated for the testing dedicated admins.  |
| REALM                     | string  | Optional | testing-idp    | Set the name of the realm in side cluster sso                               |
| REALM_DISPLAY_NAME        | string  | Optional | Testing IDP    | Realm display name in side cluster sso                                      |
| INSTALLATION_PREFIX       | string  | Optional | _None_         | If empty, the value is gotten for the the cluster using `oc get RHMIs --all-namespaces -o (pipe) jq -r .items[0].spec.namespacePrefix` |
| ADMIN_USERNAME            | string  | Optional | customer-admin | Username prefix for dedicated admins                                        |
| NUM_ADMIN                 | int     | Optional | 3              | Number of dedicated admins to be set up                                     |
| REGULAR_USERNAME          | string  | Optional | test-user      | Username prefix for regular test users                                      |
| NUM_REGULAR_USER          | int     | Optional | 10             | Number of regular user to be used.                                          |

### Configuring Github OAuth

*Note:* Following steps are only valid for OCP4 environments and will not work on OSD due to the Oauth resource being periodically reset by Hive.

Follow [docs](https://docs.openshift.com/container-platform/4.1/authentication/identity_providers/configuring-github-identity-provider.html#identity-provider-registering-github_configuring-github-identity-provider) on how to register a new Github Oauth application and add the necessary authorization callback URL for your cluster as outlined below:

```
https://oauth-openshift.apps.<cluster-name>.<cluster-domain>/oauth2callback/github
```

Once the Oauth application has been registered, navigate to the Openshift console and complete the following steps:

*Note:* These steps need to be performed by a cluster admin

- Select the `Search` option in the left-hand nav of the console and select `Oauth` from the "Resources" dropdown
- A single Oauth resource should exist named `cluster`, click into this resource
- Scroll to the bottom of the console and select the `Github` option from the `add` dropdown
- Next, add the `Client ID` and `Client Secret` of the registered Github Oauth application
- Ensure that the Github organization from where the Oauth application was created is specified in the Organization field
- Once happy that all necessary configurations have been added, click the `Add` button
- For the validation purposes, log into the Openshift console from another browser and check that the Github IDP is listed on the login screen

## Set up dedicated admins

To setup your cluster to have dedicated admins run the `./scripts/setup-htpass-idp.sh` script which creates htpasswd identity provider and creates users.

## Tests

### Unit tests

Running unit tests:
```sh
make test/unit
```

### E2E tests

A `BYPASS_STORAGE_TYPE_CHECK=true` flag is used to allow test to run when the operator is installed using cluster storage.
This may cause side effects related to the cloud resources test.

To run E2E tests against a clean OpenShift cluster using operator-sdk, build and push an image 
to your own quay repo, then run the command below changing the installation type based on which type you are testing:
```
make test/e2e INSTALLATION_TYPE=<managed/managed-api/multitenant-managed-api> OPERATOR_IMAGE=<your/repo/image:tag>
```

To run E2E tests against an existing RHMI cluster:
```
make test/functional
```

To run a single E2E test against a running cluster run the command below where E03 is the start of the test description:
```
INSTALLATION_TYPE=<managed/managed-api/multitenant-managed-api> TEST=E03 make test/e2e/single
```
### Product tests

To run products tests against an existing RHMI cluster:
```
make test/products/local
```

## Uninstalling RHOAM
This section covers uninstallation of RHOAM if it was installed via locally, OLM or on ROSA

### Local and OLM installation type
If you installed RHOAM locally or through a catalog source then you can uninstall one of two ways:

A) Create a configmap and add a deletion label (Prefered way of uninstallation).
```sh 
oc create configmap managed-api-service -n redhat-rhoam-operator
oc label configmap managed-api-service api.openshift.com/addon-managed-api-service-delete=true -n redhat-rhoam-operator
```

B) Delete the RHOAM cr.
```sh 
oc delete rhmi rhoam -n redhat-rhoam-operator
```

In both scenarios wait until the RHOAM cr is removed and then run the following command to delete the namespace.
```sh 
oc delete namespace redhat-rhoam-operator
```

#### Note: After uninstalling RHOAM you should clean up the cluster by running the following command.
```sh
export INSTALLATION_TYPE=managed-api
make cluster/cleanup && make cluster/cleanup/crds
```

### Addon
  If you installed RHOAM as an addon then you can uninstall it through the ui as shown in the picture below , or alternatively  you can run the following command. 
```sh
ocm delete /api/clusters_mgmt/v1/clusters/${clusterId}/addons/managed-api-service
```
![Uninstall RHOAM addon](https://user-images.githubusercontent.com/74991829/153239383-52edb7d5-f03a-4b1e-83ca-e5961b2ba577.png)


### ROSA Addon
  If you installed RHOAM as an addon on [ROSA](https://cloud.redhat.com/products/amazon-openshift) then you can uninstall it by running the following command.
```sh 
rosa uninstall addon \
--cluster=${clusterName} managed-api-service -y
```

## Release

See the [release doc](./RELEASE.md).
