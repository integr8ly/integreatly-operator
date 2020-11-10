# Scripts
## prepare-release.sh
This script creates a release csv for the operator under the `olm-catalog` using `operator-sdk generate csv`

**Required System Variables**

- SEMVER -> valid x.y.z format. Usage: `SEMVER=2.6.0`

**Optional System Variables**

- OLM_TYPE -> Which resource the release csv will be created under. Current Default: `integreatly-operator`.  Usage: `OLM_TYPE=managed-api-service`
- ORG -> The quay.io org to where the images will be pushed. Setting the ORG will change the image locations specified in \<olm_type>.\<version>.clusterserviceversion.yaml under `containerImage` and `image`. Default: `integreatly`. Usage: `ORG=<user-repo>`.

**Usage**

`SEMVER=2.6.0 OLM_TYPE=managed-api-service ORG=<user-repo> ./prepare-release.sh`

## rhoam-manifest-generator.sh
This script generates a manifest of the integreatly-operator which includes the direct and indirect dependencies 
used. The generated manifest corresponds to a released version and is located in: "../rhoam-manifests/" and can be used
by external entities.

**Usage**

`make manifest/release`