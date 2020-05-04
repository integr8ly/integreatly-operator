#!/usr/bin/env bash
set -e
set -o pipefail

PREVIOUS_VERSION=$(grep integreatly-operator deploy/olm-catalog/integreatly-operator/integreatly-operator.package.yaml | awk -F v '{print $2}')

create_new_csv() {
  operator-sdk generate csv --csv-version "$VERSION" --default-channel --operator-name integreatly-operator --csv-channel=rhmi --update-crds --from-version "$PREVIOUS_VERSION"
}

set_version() {
  "${SED_INLINE[@]}" "s/$PREVIOUS_VERSION/$VERSION/g" Makefile
  "${SED_INLINE[@]}" "s/$PREVIOUS_VERSION/$VERSION/g" version/version.go
}

set_images() {
  : "${IMAGE_TAG:=v${SEMVER}}"
  "${SED_INLINE[@]}" "s/image:.*/image: quay\.io\/integreatly\/integreatly-operator:$IMAGE_TAG/g" "deploy/olm-catalog/integreatly-operator/${VERSION}/integreatly-operator.v${VERSION}.clusterserviceversion.yaml"
  "${SED_INLINE[@]}" "s/containerImage:.*/containerImage: quay\.io\/integreatly\/integreatly-operator:$IMAGE_TAG/g" "deploy/olm-catalog/integreatly-operator/${VERSION}/integreatly-operator.v${VERSION}.clusterserviceversion.yaml"
}

if [[ -z "$SEMVER" ]]; then
 echo "ERROR: no SEMVER value set"
 exit 1
fi


if [[ $SEMVER =~ ^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$ ]]; then
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
fi

# Include the webhook service in the bundle (temporal solution as OLM will soon
# support webhooks as part of the CSV:
# https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/contributors/design-proposals/webhooks.md
cp deploy/webhook-service.yaml deploy/olm-catalog/integreatly-operator/$VERSION/webhook-service.yaml

set_images