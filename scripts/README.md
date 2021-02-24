# Scripts
## prepare-release.sh
This script creates a release csv for the operator under the `olm-catalog` using `operator-sdk generate csv`

**Usage**

`SEMVER=2.6.0 OLM_TYPE=managed-api-service ORG=<user-repo> ./prepare-release.sh`

## rhoam-manifest-generator.sh
This script generates a manifest of the integreatly-operator which includes the direct and indirect dependencies 
used. The generated manifest corresponds to a released version and is located in: "../rhoam-manifests/" and can be used
by external entities.

**Usage**

`make manifest/release`



### System variables

| Variable |                      Format                    |     Type     |        Default         | Details |
|----------|:----------------------------------------------:|:------------:|:----------------------:|---------|
| SEMVER   | `<x.y.z>`                                      | **Required** |  n/a                   | Release version of `OLM_TYPE`. Example: `SEMVER=2.6.0` |
| OLM_TYPE | `integreatly-operator`or `managed-api-service` | Optional     | `integreatly-operator` | Which resource the release csv will be created under. |
| REG      | `<registry>`                                   | Optional     | `quay.io`              | The `BUILD_TOOL` registry where the bundles/indices that package the operator are stored.
| ORG      | `<user-repo>`                                  | Optional     | `integreatly`          | The organization/user in the registry that publishes the bundles. Where the images will be pushed to. Setting the ORG will change the image locations specified in `<olm_type>.<version>.clusterserviceversion.yaml` under containerImage and image. |
| CHANNEL  | `stable` or `alpha`                            | Optional     | `alpha`                | Allows to deliver different releases of an operator, so we can have a `stable` channel or an `alpha` channel with more up to date releases.
|BUILD_TOOL| `docker` or `podman`                           | Optional     | `docker`               | The tool to build the images of the bundles and indices.
