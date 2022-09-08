# OLM installation

As OperatorSource is being deprecated and the bundle format will become the
standard packaging for operator, this document describes how to install the RHMI
operator through OLM, by creating bundles and indices and a CatalogSource. Mainly
for development/testing purposes.

## Nomenclature

* _Bundle_: A bundle is a non-runnable docker image containing the operator 
  manifests for a specific release.
* _Index_: An index is a docker image exposing a database throgh GRPc, which
  contains references to many bundles
* _OLM_TYPE_: managed-api-service for RHOAM, integreatly-operator for RHMI

> **Both bundles and indices are potentially pulled by the cluster, so they must be made public in order to successfully perform the installation**

## Prerequites

* `opm` is a CLI tool used to automate the generation of bundles and indices.
  * Information on how to build it can be found [here](https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md#opm-operator-package-manager)
  * Releases page for direct download: https://github.com/operator-framework/operator-registry/releases
* You need to be logged to your registry.


## Steps

There are 2 approaches on building the bundles and indexes for OLM installation. It is important to know, that if you need your custom build RHOAM or RHMI binary, your will need to build the image yourself and update it in the CSV in the `containerImage` and `image` for a given version.
To build your own RHOAM or RHMI image run:

`INSTALLATION_TYPE=<managed-api|managed> ORG=<organization> make image/build`

Make command above can take in a couple of parameters but the above is the minimum you should provide:
`REG` - your registry, for example: quay.io, defaults to quay.io
`ORG` - organisation within your registry, defaults to integreatly
`PROJECT` - defaults to based on OLM TYPE - managed-api-service|integreatly-operator
`TAG` - image tag, defaults to RHOAM_TAG which is the most recent version

Example:
`INSTALLATION_TYPE=managed-api ORG=<quay username> TAG=master make image/build`
The above will build an image of RHOAM with a tag `master`

Next, please push the image to your registry with:
`INSTALLATION_TYPE=managed-api ORG=<quay username> TAG=master make image/push`

Once the image has been pushed, ensure it's available to public and replace both fields mentioned above in the CSV.


## Automated approach

In integreatly operator repository use the `make create/olm/bundle` command with the following parameters:
>OLM_TYPE - must be specified, refers to type of operator lifecycle manager type, can either be integreatly-operator (RHMI) or managed-api-service (RHOAM)

>UPGRADE - defaults to false, if upgrade is false the oldest version specified in the BUNDLE_VERSIONS will have it's replaces removed, otherwise, replaces will stay

>BUNDLE_VERSIONS - specifies the versions that are going to have the bundles build. Versions must exists in the bundle/OLM_TYPE folder and must be listed in a descending order

>ORG - organisation of where to push the bundles and indexes (for quay.io it is the organisation)

>REG - registry of where to push the bundles and indexes, defaults to quay.io

>BUILD_TOOL - tool used for building the index, defaults to docker - for now, only podman and docker are verified as the supported build tools

>OC_INSTALL - set to true if you want the catalogue source to be created pointing to the "oldest" version within the versions specified (version must have no replaces field)(must be oc logged in)

Example:
```
make create/olm/bundle OLM_TYPE=managed-api-service UPGRADE=false BUNDLE_VERSIONS=1.16.0,1.15.2 ORG=<YOUR_QUAY_USERNAME> OC_INSTALL=true
```
Running the above command will:
1. Remove `replaces` field from 1.15.2 making it the initial version to be installed on the cluster
2. Build a bundle of 1.15.2 and 1.16.0 folder and push it to quay.io
3. Build an index for 1.15.2 and 1.16.0 and push it to quay.io
4. OC create catalogue source pointing to 1.15.2

The make command can also be used for a single bundle and index, or 2 releases.  


## Manual approach

For each release that we want to make available in OLM, we need to generate a bundle.
The bundles are stored in `bundles/<OLM_TYPE>/<VERSION>` (example: `integreatly-operator/bundles/managed-api-service/1.15.0`), having
one directory per release.

### Generate new bundle

In order to generate a new release, the `prepare-release` script can be used:

```sh
OLM_TYPE=<OLM_TYPE> SEMVER=<release version> ORG=<quay username> make release/prepare
```

This will generate a new directory with a new CSV and up to date manifests.
Ensure the following in the newly generated CSV:

1. The image references the operator image we want to use
2. If this is the earliest CSV we want to make available, manually delete the
  `replaces` field, as otherwise it'll fail to validate the bundle when creating 
  an index.

### Build bundle for a given release

From main folder, run:
```
make olm/bundle BUNDLE_TAG=quay.io/<quay username>/<OLM_TYPE>-bundle:<release version> VERSION=<VERSION> OLM_TYPE=<OLM_TYPE>
```
Example:
```
make olm/bundle BUNDLE_TAG="quay.io/<quay username>/managed-api:1.15.2" VERSION=1.15.2 OLM_TYPE=managed-api-service
```
The above command will build the bundle for you of existing bundle files in the ./bundles/<OLM_TYPE>/<VERSION> folder.

> **It's important to have the CSV in a state that you need, for example, if this is the initial bundle that will be installed on cluster, remove the replaces field, if it's to use your own RHOAM binary, update the containerImage and image fields in the CSV.**

> **If there are old bundle files in this directory you will need to remove those before re-running the bundle command.**

Next, push the bundle:

```sh
docker push quay.io/<quay username>/<OLM_TYPE>-bundle:<release version>
operator-sdk bundle validate quay.io/<quay username>/<OLM_TYPE>-bundle:<release version>
```
Example:
```
docker push quay.io/<quay username>/managed-api-service-bundle:1.15.2
operator-sdk bundle validate quay.io/<quay username>/managed-api-service-bundle:1.15.2
```

### Generate index for a given release

`opm` includes tooling to generate an index for many bundles. Run the command:

```sh
opm index add \
    --bundles quay.io/<your org>/<OLM_TYPE>-bundle:<bundle 1 version> \
    --bundles quay.io/<your org>/<OLM_TYPE>-bundle:<bundle n version if more than one bundle in the bundle chain> \
    --build-tool docker \
    --tag quay.io/<your org>/<OLM_TYPE>-index:<index version>
```
Example:
```
opm index add \
--bundles quay.io/<quay username>/managed-api-service-bundle:1.15.2 \
--bundles quay.io/<quay username>/managed-api-service-bundle:1.16.0 \
--bundles quay.io/<quay username>/managed-api-service-bundle:1.17.0 \
--tag quay.io/<quay username>/managed-api-service-index:1.17.0
```

The following parameters have been used:

* `--bundles`: Specifies the tag of a bundle to include in the index. Can be
  passed many times, as many as bundles we want to include
* `--build-tool`: Tool to build the image. Can be docker or podman
* `--tag`: Tag of the index image. For convenience, use the version of the latest
  release included in the index

Push the index

```sh
docker push quay.io/<your org>/<OLM_TYPE>-index:<index version>
```
Example:
```
docker push quay.io/<quay username>/managed-api-service-index:1.17.0
```

### Create CatalogSource

Once we have an index image, all we have to do is create a CatalogSource referencing
that image. This will prompt the generation of a pod and a service exposing
the bundles, which will be discovered and aggregated by OLM, generating the PackageManifest
that we'll subscribe to.

Create the following CatalogSource

> Save the YAML below to a file `rhmi-operators.yaml` and run `oc apply -f rhmi-operators.yaml`

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: rhmi-operators
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/<your org>/<OLM_TYPE>-index:<index version>
```

### Prepare cluster

Run the following command to prepare the cluster:

> This will create the `redhat-rhmi-operator` or `redhat-rhoam-operator` namespace/project as well as the secrets required for the installation.

```sh
INSTALLATION_TYPE=<managed-api||managed> LOCAL=false make cluster/prepare/local
```

### Deploy RHMI CR
  Installation of RHOAM through OLM will not deploy the RHMI CR automatically. Run the following command to deploy the RHMI CR to the cluster:
```sh 
INSTALLATION_TYPE=managed-api LOCAL=false make deploy/integreatly-rhmi-cr.yml
```

## Install from OperatorHub

Once OLM has created the PackageManifest, find the `RHOAM` or `RHMI` in the operator hub and
click install


## Scenario: Simulating upgrades

In order to simulate an upgrade, a separate index can be created, containing an
additional release that replaces the current one. Push this index and, when
ready to make the upgrade available, update the CatalogSource in the spec to
reference the new index.
