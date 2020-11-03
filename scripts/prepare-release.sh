#!/usr/bin/env bash
set -e
set -o pipefail

if [[ -z "$OLM_TYPE" ]]; then
  OLM_TYPE="integreatly-operator"
fi

case $OLM_TYPE in
  "integreatly-operator")
    PREVIOUS_VERSION=$(grep $OLM_TYPE deploy/olm-catalog/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $2}') || echo "No previous version"
    ;;
  "managed-api-service")
    PREVIOUS_VERSION=$(grep $OLM_TYPE deploy/olm-catalog/$OLM_TYPE/$OLM_TYPE.package.yaml | awk -F v '{print $3}') || echo "No previous version"
    ;;
  *)
    echo "Invalid OLM_TYPE set"
    echo "Use \"integreatly-operator\" or \"managed-api-service\""
    exit 1
    ;;
esac

if [[ -z "$ORG" ]]; then
  ORG="integreatly"
else
  ORG="$ORG"
fi

create_new_csv() {

  if [[ -z "$PREVIOUS_VERSION" ]]
    then
      operator-sdk generate csv --csv-version "$VERSION" --default-channel --operator-name "$OLM_TYPE" --csv-channel=rhmi --update-crds
    else
      operator-sdk generate csv --csv-version "$VERSION" --default-channel --operator-name "$OLM_TYPE" --csv-channel=rhmi --update-crds --from-version "$PREVIOUS_VERSION"
  fi
}

update_csv() {
  operator-sdk generate csv --csv-version "$VERSION" --default-channel --operator-name "$OLM_TYPE" --csv-channel=rhmi --update-crds

}

set_version() {
  if [[ -z "$PREVIOUS_VERSION" ]]
    then
      echo "No previous version please set correct values in the Makefile and version/version.go files"
    else
      case $OLM_TYPE in
        "integreatly-operator")
          "${SED_INLINE[@]}" -E "s/RHMI_TAG\s+\?=\s+$PREVIOUS_VERSION/RHMI_TAG \?= $VERSION/g" Makefile
          "${SED_INLINE[@]}" -E "s/version\s+=\s+\"$PREVIOUS_VERSION\"/version = \"$VERSION\"/g" version/version.go
          ;;
        "managed-api-service")
          "${SED_INLINE[@]}" -E "s/RHOAM_TAG\s+\?=\s+$PREVIOUS_VERSION/RHOAM_TAG \?= $VERSION/g" Makefile
          "${SED_INLINE[@]}" -E "s/managedAPIVersion\s+=\s+\"$PREVIOUS_VERSION\"/managedAPIVersion = \"$VERSION\"/g" version/version.go
          ;;
        *)
          echo "No version found for install type : $(OLM_TYPE)"
          ;;
      esac
  fi
}

set_images() {
  : "${IMAGE_TAG:=v${SEMVER}}"
  "${SED_INLINE[@]}" "s/image:.*/image: quay\.io\/$ORG\/$OLM_TYPE:$IMAGE_TAG/g" "deploy/olm-catalog/$OLM_TYPE/${VERSION}/$OLM_TYPE.v${VERSION}.clusterserviceversion.yaml"
  "${SED_INLINE[@]}" "s/containerImage:.*/containerImage: quay\.io\/$ORG\/$OLM_TYPE:$IMAGE_TAG/g" "deploy/olm-catalog/$OLM_TYPE/${VERSION}/$OLM_TYPE.v${VERSION}.clusterserviceversion.yaml"
}

set_csv_not_service_affecting() {
  echo "Update CSV for release $SEMVER to be not service affecting"
  yq w -i "deploy/olm-catalog/$OLM_TYPE/${VERSION}/$OLM_TYPE.v${VERSION}.clusterserviceversion.yaml" --tag '!!str' metadata.annotations.serviceAffecting "false"
}

if [[ -z "$SEMVER" ]]; then
 echo "ERROR: no SEMVER value set"
 exit 1
fi


if [[ $SEMVER =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$ ]]; then
  echo "Valid version string: ${SEMVER}"
else
  echo "Error: Invalid version string: ${SEMVER}"
  exit 1
fi

VERSION=$(echo "$SEMVER" | awk -F - '{print $1}')

# Set sed -i as it's different for mac vs gnu
if [[ $(uname) = Darwin ]]; then
  SED_INLINE=(sed -i '')
else
  SED_INLINE=(sed -i)
fi

# We have a new version so generate the csv
if [[ "$VERSION" != "$PREVIOUS_VERSION" ]]; then
  create_new_csv
  set_version
else
  update_csv
fi

set_images

if [[ ! -z "$NON_SERVICE_AFFECTING" ]]; then
 set_csv_not_service_affecting
fi
