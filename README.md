# Integreatly Operator

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

## Deploying to a Cluster with OLM
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

To run E2E tests against a clean OpenShift cluster using operator-sdk:
```
make test/e2e
```

To run E2E tests against an existing RHMI cluster:
```
make test/functional
```

## Using `ocm` for installation of RHMI

If you want to test your changes on a cluster, the easiest solution would be to spin up OSD 4 cluster using `ocm`. If you want to spin up a cluster using BYOC (your own AWS credentials), follow the additional steps marked as **BYOC**.

#### Prerequisites
* [OCM CLI](https://github.com/openshift-online/ocm-cli/releases)
* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)
* [jq](https://stedolan.github.io/jq/)

#### Steps

1. Download the CLI tool and add it to your PATH
2. Export [OCM_TOKEN](https://github.com/openshift-online/ocm-cli#log-in): `export OCM_TOKEN="<TOKEN_VALUE>"`
3. Login via OCM: 
```
make ocm/login
```

**BYOC**
Make sure you have credentials for IAM user with admin access to AWS and other IAM user called "osdCcsAdmin" created in AWS, also with admin access.
Export the credentials for your IAM user, set BYOC variable to `true` and create a new access key for "osdCcsAdmin" user:
```
export AWS_ACCOUNT_ID=<REPLACE_ME>
export AWS_ACCESS_KEY_ID=<REPLACE_ME>
export AWS_SECRET_ACCESS_KEY=<REPLACE_ME>
export BYOC=true
make ocm/aws/create_access_key
```

4. Create cluster template: `make ocm/cluster.json`.

This command will generate `ocm/cluster.json` file with generated cluster name. This file will be used as a template to create your cluster via OCM CLI.
By default, it will set the expiration timestamp for a cluster for 4 hours, meaning your cluster will be automatically deleted after 4 hours after you generated this template. If you want to change the default timestamp, you can update it in `ocm/cluster.json` or delete the whole line from the file if you don't want your cluster to be deleted automatically at all. 

5. Create the cluster: `make ocm/cluster/create`.

This command will send a request to [Red Hat OpenShift Cluster Manager](https://cloud.redhat.com/) to spin up your cluster and waits until it's ready. You can see the details of your cluster in `ocm/cluster-details.json` file

6. Once your cluster is ready, OpenShift Console URL will be printed out together with the `kubeadmin` user & password. These are also saved to `ocm/cluster-credentials.json` file. Also there will be `ocm/cluster.kubeconfig` file created that you can use for running `oc` commands right away, for example, for listing all projects on your OpenShift cluster:

```
oc --config ocm/cluster.kubeconfig projects
```

7. If you want to install the latest released RHMI, you can trigger it by applying an RHMI addon.
Run `make ocm/install/rhmi-addon` to trigger the installation. Once the installation is completed, the installation CR with RHMI components info will be printed to the console.

8. If you want to delete your cluster, run `make ocm/cluster/delete`
**BYOC**

## Release

- Create `release-<release-number>` branch
- Update `tag` and `previoustag` in makefile
- Run `make gen/csv`
- - Perform any manual tidying up of the generated CSV as required.
- Run `make gen/namespaces` against a fully installed cluster. 
- Make a PR against this repo
- Get a review on PR and see e2e test pass
- Wait for merge
- Run `make image/build/push REPO=integreatly`
- Run `make push/csv REPO=integreatly` (doesn’t affect managed-tenants)
- Make <release-number> tag on `release-<release-number>` branch and push it to integr8ly/integreatly-operator repo
- Make a release in github UI
- Tell QE, so they can update pipelines to new release-number
- Take CSV files from deploy/olm-catalog and make a PR to managed-tenants, make any changes as required beforehand.
- Build the [test harness image](https://github.com/integr8ly/integreatly-operator/tree/master/test#build-functional-test-image), using the same release tag + the tag `latest-staging` and push it to quay.io
