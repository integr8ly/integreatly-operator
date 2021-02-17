# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling managed products.

An Integreatly Operator can be installed using two different flavours: `managed` or `managed-api`

To switch between the two you can use export the `INSTALLATION_TYPE` env or use it in conjunction with any of the make commands referenced in this README

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


## Prerequisites

- [operator-sdk](https://github.com/operator-framework/operator-sdk) version v1.2.0.
- [go](https://golang.org/dl/) version 1.13.4+
- [moq](https://github.com/matryer/moq)
- [oc](https://docs.okd.io/latest/cli_reference/openshift_cli/getting-started-cli.html) version v4.6+
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
- RHOAM (managed-api): 18 vCPU. More details can be found in the [service definition](https://access.redhat.com/articles/5534341) 
  under the "Resource Requirements" section

Consider using IN_PROW [optional variable](#variables-table) in case of limited resources.  
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
Include the `INSTALLATION_TYPE`. See [here](#variables-table) about this and other optional configuration variables.
```shell
INSTALLATION_TYPE=<managed/managed-api> make cluster/prepare/local
```

### 3. Run integreatly-operator
Include the `INSTALLATION_TYPE` if you haven't already exported it. 
The operator can now be run locally:
```shell
INSTALLATION_TYPE=<managed/managed-api> make code/run
```

*Note:* if the operator doesn't find an RHMI cr, it will create one (Name: `rhmi/rhoam`).

### 4. Validate installation 

Use following commands to validate that installation succeeded:

For `RHMI` (managed): `oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq .status.stage`

For `RHOAM` (managed-api): `oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage `

Once the installation completed the command wil result in following output:  
```yaml
"complete"
```

### Variables table 
In case you desire to use optional variables, consider running the following command before `make code/run` to ensure that the variables change
would be implemented: `<VARIABLE>=<value> make deploy/integreatly-rhmi-cr.yml`

| Variable | Options | Type | Default | Details |
|----------|---------|:----:|---------|-------|
| INSTALLATION_TYPE     | `managed` or `managed-api`| **Required** |`managed`  | Manages installation type. `managed` stands for RHMI. `managed-api` for RHOAM. |
| IN_PROW               | `true` or `false`         | Optional      |`false`    | If `true`, reduces the number of pods created. Use for small clusters |
| USE_CLUSTER_STORAGE   | `true` or `false`         | Optional      |`true`     | If `true`, installs application to the cloud provider. Otherwise installs to the OpenShift. |

## Deploying to a Cluster with OLM and the Bundle Format

### 1. Bundles
There exists a number of variables, that can prepend the make target below. Refer to [this](/scripts/README.md#system-variables) document.


To generate bundles run the script: `./scripts/bundle-rhmi-opertors.sh `

### 2. Install from OperatorHub
OLM will create a PackageManifest (integreatly) based on the CatalogSource (rhmi-operators) in the openshift-marketplace namespace. 
Confirm both and then find the RHMI in the OperatorHub. Verify that the version references the latest version available in the index and click install

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

Tip: set a `PASSWORD` env var to define a password for the users. A random password is generated when this env var is not set.

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

To run E2E tests against a clean OpenShift cluster using operator-sdk, build and push an image 
to your own quay repo, then run the command below changing the installation type based on which type you are testing:
```
make test/e2e INSTALLATION_TYPE=<managed/managed-api> OPERATOR_IMAGE=<your/repo/image:tag>
```

To run E2E tests against an existing RHMI cluster:
```
make test/functional
```

To run a single E2E test against a running cluster run the command below where E03 is the start of the test description:
```
go clean -testcache && go test -v ./test/functional -run="//^E03" -timeout=80m
```
### Product tests

To run products tests against an existing RHMI cluster:
```
make test/products/local
```

## Release

See the [release doc](./RELEASE.md).
