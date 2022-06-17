# Scripts
## prepare-release.sh
This script creates a release csv for the operator under the `olm-catalog` using `operator-sdk generate csv`

**Usage**

`SEMVER=1.23.0 OLM_TYPE=managed-api-service ORG=<user-repo> ./prepare-release.sh`

### System variables

| Variable |                                       Format                                        |     Type     |        Default        | Details                                                                                                                                                                                                                                              |
|----------|:-----------------------------------------------------------------------------------:|:------------:|:---------------------:|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| SEMVER   |                                      `<x.y.z>`                                      | **Required** |          n/a          | Release version of `OLM_TYPE`. Example: `SEMVER=1.23.0`                                                                                                                                                                                              |
| OLM_TYPE | `managed-api-service` or `multitenant-managed-api-service` | Optional     | `managed-api-service` | Which resource the release csv will be created under.                                                                                                                                                                                                |
| REG      |                                    `<registry>`                                     | Optional     |       `quay.io`       | The `BUILD_TOOL` registry where the bundles/indices that package the operator are stored.                                                                                                                                                            |
| ORG      |                                    `<user-repo>`                                    | Optional     |     `integreatly`     | The organization/user in the registry that publishes the bundles. Where the images will be pushed to. Setting the ORG will change the image locations specified in `<olm_type>.<version>.clusterserviceversion.yaml` under containerImage and image. |
| CHANNEL  |                                 `stable` or `alpha`                                 | Optional     |        `alpha`        | Allows to deliver different releases of an operator, so we can have a `stable` channel or an `alpha` channel with more up to date releases.                                                                                                          |
|BUILD_TOOL|                                `docker` or `podman`                                 | Optional     |       `docker`        | The tool to build the images of the bundles and indices.                                                                                                                                                                                             |
