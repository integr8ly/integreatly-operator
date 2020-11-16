# Integreatly Operator

---
**NOTE**

When working on `RHOAM`, all `make` commands should be prefixed with `INSTALLATION_TYPE=managed-api`.
This will configure any variables that may be required.

The default `INSTALLATION_TYPE` is `managed` which configures variables for use with `RHMI`

---

A Kubernetes Operator based on the Operator SDK for installing and reconciling Integreatly products.

### Project status: _alpha_

This is a proof of concept/alpha version. Most functionality is present but it is highly likely there are bugs and improvements needed.

### Installed products
Currently the operator installs the following products:
- AMQ Online
- AMQ Streams
- Codeready
- Fuse
- Nexus
- RHSSO (both a cluster instance and a user instance)
- 3scale
- Integreatly solution explorer

## Prerequisites

NOTE: Due to a change in how networking is configured for openshift in v4.4.6 (mentioned in the [cloud resource operator](https://github.com/integr8ly/cloud-resource-operator#supported-openshift-versions)) there is a limitation on the version of Openshift that RHMI can be installed on for BYOC clusters.
Due to this change the use of integreatly-operator <= v2.4.0 on Openshift >= v4.4.6 is unsupported. Please use >= v2.5.0 of integreatly-operator for Openshift >= v4.4.6.

- [operator-sdk](https://github.com/operator-framework/operator-sdk) version v0.15.1.
- [go](https://golang.org/dl/) version 1.13.4+
- [moq](https://github.com/matryer/moq)
- [oc](https://docs.okd.io/3.11/cli_reference/get_started_cli.html#cli-reference-get-started-cli) version v3.11+
- Access to an Openshift v4.2.0+ cluster
- A user with administrative privileges in the OpenShift cluster
- AWS account with permissions to create S3 buckets

After installation, the following commands must be run to avoid a known [issue](https://github.com/matryer/moq/issues/98) related to the Moq package:
```
make code/compile
go install github.com/matryer/moq
```

## Local Setup

Download the integreatly-operator project:
```sh
mkdir -p $GOPATH/src/github.com/integr8ly
cd $GOPATH/src/github.com/integr8ly
git clone https://github.com/integr8ly/integreatly-operator
cd integreatly-operator
```

If the cluster is not already prepared for the integreatly-operator, you will need to do the following:
```
make cluster/prepare/project
make cluster/prepare/crd
make cluster/prepare/smtp
```

* 3scale requires AWS S3 bucket credentials for storage. The bucket should have all public access turned off.

Currently this secret (`threescale-blobstorage-<installation-name>`) is created with dummy credentials by the [cloud resource operator](https://github.com/integr8ly/cloud-resource-operator), in the namespace the integreatly operator is deployed into. In order for this feature to work, these credentials should be replaced:
    * _bucketName_: The name of the AWS bucket
    * _bucketRegion_: The AWS region where the bucket has been created
    * _credentialKeyID_: The AWS access key
    * _credentialSecretKey_: The AWS secret key

You can use this command to replace S3 credentials in 3Scale secret:
```sh
oc process -f deploy/s3-secret.yaml -p AWS_ACCESS_KEY_ID=<YOURID> -p AWS_SECRET_ACCESS_KEY=<YOURKEY> -p AWS_BUCKET=<YOURBUCKET> -p AWS_REGION=eu-west-1 -p NAMESPACE=<integreatly-operator-namespace> -p NAME=threescale-blobstorage-<installation-name> | oc replace -f -
```

* Backup jobs require AWS S3 bucket credentials for storage. A `backups-s3-credentials` Secret is created the same way as a 3Scale secret described above.

You can use this command to replace S3 credentials in backup secret:
```sh
oc process -f deploy/s3-secret.yaml -p AWS_ACCESS_KEY_ID=<YOURID> -p AWS_SECRET_ACCESS_KEY=<YOURKEY> -p AWS_BUCKET=<YOURBUCKET> -p AWS_REGION=eu-west-1 -p NAMESPACE=<integreatly-operator-namespace> | oc replace -f -
```

### RHMI custom resource
An `RHMI` custom resource can now be created which will kick of the installation of the integreatly products, once the operator is running:
```sh
# Create the installation custom resource
oc create -f deploy/crds/examples/rhmi.cr.yaml

# The operator can now be run locally
make code/run
```
*Note:* if an operator doesn't find RHMI resource, it will create one (Name: `rhmi`).

### Logging in to SSO

In the OpenShift UI, in `Projects > redhat-rhmi-rhsso > Networking > Routes`, select the `sso` route to open up the SSO login page.

# Bootstrap the project

```sh
make cluster/prepare/local
```

### Configuring Github OAuth

*Note:* The following steps are only valid for OCP4 environments and will not work on OSD due to the Oauth resource being periodically reset by Hive.

Follow [docs](https://docs.openshift.com/container-platform/4.1/authentication/identity_providers/configuring-github-identity-provider.html#identity-provider-registering-github_configuring-github-identity-provider) on how to register a new Github Oauth application and add the necessary authorization callback URL for your cluster as outlined below:

```
https://oauth-openshift.apps.<cluster-name>.<cluster-domain>/oauth2callback/github
```

Once the Oauth application has been registered, navigate to the Openshift console and complete the following steps:

*Note:* These steps need to be performed by a cluster admin

- Select the `Search` option in the left hand nav of the console and select `Oauth` from the dropdown
- A single Oauth resource should exist named `cluster`, click into this resource
- Scroll to the bottom of the console and select the `Github` option from the `add` dropdown
- Next, add the `Client ID` and `Client Secret` of the registered Github Oauth application
- Ensure that the Github organization from where the Oauth application was created is specified in the Organization field
- Once happy that all necessary configurations have been added, click the `Add` button
- For validation purposes, log into the Openshift console from another browser and check that the Github IDP is listed on the login screen

## Deploying to a Cluster with OLM and the Bundle Format

This deployment approach uses a CatalogSource which references an index image. The index
image contains references to image bundles which specify the specific versions of the RHMI operator. 

### Nomenclature

* _Bundle_: A bundle is a non-runnable docker image containing the operator 
  manifests for a specific release.
* _Index_: An index is a docker image exposing a database throgh GRPc, which
  contains references to many bundles

> **Both bundles and indices are potentially pulled by the cluster, so they must be made public in order to successfully perform the installation**


### Prerequisites
* `opm` is a CLI tool used to automate the generation of bundles and indices.
  * Information on how to build it can be found [here](https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md#opm-operator-package-manager)
  * Releases page for direct download: https://github.com/operator-framework/operator-registry/releases
* Bundle validation requires operator-sdk >= 0.18.2 and above

Make sure to export the variables above (see [local setup](#local-setup)), then run:

```sh
make cluster/prepare/bundle
```

For local development, update the ORG and ensure the repositories are publicly accessible.
Potentially, 3 repositories need to be publicly available, 
* \<REG>/\<ORG>/integreatly-operator
* \<REG>/\<ORG>/integreatly-index
* \<REG>/\<ORG>/integreatly-bundle 

### Variables
The following variables can prepend the make target below 
* CHANNEL, default alpha
* ORG, default integreatly
* REG, default quay.io
* BUILD_TOOL, default docker

### Deployment Scenarios

#### Deploy the latest bundle version 
This assumes no prior installation of the RHMI operator. As such, it will remove the replaces field in the CSV file to 
prevent the attempted replacement of an existing version of the operator available for installation.
```sh
ORG=<YOUR_ORG> make install/olm/bundle 
```

#### Deploy a specific bundle version
This will assume not upgrading and remove the replaces field from CSV 2.5.0
```sh
ORG=<YOUR_ORG> BUNDLE_VERSIONS="2.5.0" make install/olm/bundle
```

#### Deploy an upgrade version
Example shows upgrade from 2.5.0 to 2.4.0. 2.4.0 must already be installed on the cluster.
2.5.0 must reference 2.4.0 in the CSV replaces field 
```sh
ORG=<YOUR_ORG> BUNDLE_VERSIONS="2.5.0,2.4.0" UPGRADE=true make install/olm/bundle 
```

#### Create a new bundle and deploy
The SEMVER version should be a logical SEMVER increment of the existing lastest bundle. 
The latest bundle will be copied and used as a reference for this new release.  
**NOTE:** If creating a new operator image in your repository, you need to update the CSV with references
to your image.
```sh
ORG=<YOUR_ORG> SEMVER=<X.Y.Z> make release/prepare
ORG=<YOUR_ORG> TAG=<X.Y.Z> make image/build/push    (create a new operator image if required)
ORG=<YOUR_ORG> make install/olm/bundle 
```
**NOTE:** If creating a new version to replace an existing i.e. upgrade, add both versions to the BUNDLE_VERSIONS VARIABLE
```sh
ORG=<YOUR_ORG> BUNDLE_VERSIONS="<NEW-SEMVER>,<OLD-SEMVER>" make install/olm/bundle 
```



### Install from OperatorHub
OLM will created a PackageManifest (integreatly) based on the CatalogSource (rhmi-operators) in the openshift-marketplace namespace. 
Confirm both and then find the RHMI in the OperatorHub. Verify that the version references the latest version available in the index and click install


## Deploying to a Cluster with OLM and OperatorSource (Deprecated)
Make sure to export the variables above (see [local setup](#local-setup)), then run:

```sh
make cluster/prepare
```

Within a few minutes, the Integreatly operator should be visible in the OperatorHub (`Catalog > OperatorHub`). To create a new subscription, click on the Install button, choose to install the operator in the created namespace and keep the approval strategy on automatic.

Once the subscription shows a status of `installed`, a new `RHMI` custom resource can be created which will begin to install the supported products.

In `Catalog > Developer Catalog`, choose the RHMI Installation and click create. An example RHMI CR can be found below:

```yml
apiVersion: integreatly.org/v1alpha1
kind: RHMI
metadata:
  name: example-rhmi
spec:
  type: managed
  namespacePrefix: redhat-rhmi-
  selfSignedCerts: true
  useClusterStorage: true
  smtpSecret: redhat-rhmi-smtp
  deadMansSnitchSecret: redhat-rhmi-deadmanssnitch
  pagerdutySecret: redhat-rhmi-pagerduty
```

## Set up testing IDP for OSD cluster
You can use the `scripts/setup-sso-idp.sh` script to setup a "testing-idp" realm in cluster SSO instance and add it as IDP of your OSD cluster.
With this script you will get few regular users - test-user[01-10] and few users that will be added to dedicated-admins group - customer-admin[01-03].

Prerequisites:
- `oc` command available on your machine (latest version can be downloaded [here](https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/))
- `ocm` command available ( the newest CLI can be downloaded [here](https://github.com/openshift-online/ocm-cli/releases) and you install it with `mv (your downloaded file) /usr/local/bin/ocm`) (necessary only if using OSD cluster)
- OC session with cluster admin permissions in a target cluster
- OCM session (necessary only if using OSD cluster)

Tip: set `PASSWORD` env var to define a password for the users. Random password is generated when this env var is not set.


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
to your own quay repo, then run the command below changing the installtion type based on which type you are testing:
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
### Products tests

To run products tests against an existing RHMI cluster
```
make test/products/local
```

## Using `ocm` for installation of RHMI

If you want to test your changes on a cluster, the easiest solution would be to spin up OSD 4 cluster using `ocm`.
See [here](https://github.com/integr8ly/delorean/tree/master/docs/ocm) for an up to date guide on how to do this.

## Release

See the [release doc](./RELEASE.md).