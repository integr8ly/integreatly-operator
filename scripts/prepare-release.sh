#!/usr/bin/env bash
set -e
set -o pipefail

PREVIOUS_VERSION=$(cat deploy/olm-catalog/integreatly-operator/integreatly-operator.package.yaml | grep integreatly-operator | awk -F v '{print $2}')

create_new_csv() {
  operator-sdk generate csv --csv-version "$VERSION" --default-channel --operator-name integreatly-operator --csv-channel=rhmi --update-crds --from-version "$PREVIOUS_VERSION"
}

set_version() {
  sed -i "s/$PREVIOUS_VERSION/$VERSION/g" Makefile
  sed -i "s/$PREVIOUS_VERSION/$VERSION/g" version/version.go
}

set_images() {
  sed -i "s/image:.*/image: quay\.io\/integreatly\/integreatly-operator:v$SEMVER/g" deploy/olm-catalog/integreatly-operator/${VERSION}/integreatly-operator.v${VERSION}.clusterserviceversion.yaml
  sed -i "s/containerImage:.*/containerImage: quay\.io\/integreatly\/integreatly-operator:v$SEMVER/g" deploy/olm-catalog/integreatly-operator/${VERSION}/integreatly-operator.v${VERSION}.clusterserviceversion.yaml
}

print_usage() {
  echo "-s SEMVER [1.0.0-rc1]"
  exit 1
}

if [[ -z "$SEMVER" ]]; then
 echo "ERROR: no SEMVER value set"
 print_usage
 exit 1
fi


if [[ $SEMVER =~ ^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$ ]]; then
  echo "Valid version string: ${SEMVER}"
else
  echo "Error: Invalid version string: ${SEMVER}"
  exit 1
fi

VERSION=$(echo $SEMVER | awk -F - '{print $1}')

# We have a new version so generate the csv
if [[ "$VERSION" != "$PREVIOUS_VERSION" ]]; then
  create_new_csv
  set_version
fi

set_images