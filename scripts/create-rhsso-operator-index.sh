#!/usr/bin/env bash
set -x
#
# Prereq:
# - opm
# - yq (4.24.2+)
#
# Function:
# Script creates rhsso operator index/indices for RHOAM
#
# Usage:
# REG=<REGISTRY> ORG=<QUAY ORG> IMAGE=<IMAGE NAME> BUILD_TOOL=<docker|podman>  VERSION=<VERSION TAG> make create/rhsso/index
# REG - registry for storing images, defaults is quay.io
# ORG - organization, defaults is integreatly
# IMAGE - name of the Index image to create (without version)
# VERSION - index image version (repository tag)
# BUILD_FRESH - if true - build an index image from scratch, includes all bundles beneath the max version (VERSION)
#             - otherwise - append the bundle version (VERSION) to the existing index in products/installation.yaml
# BUILD_TOOL - tool used for index image build, defaults to docker
#
# Example (using podman instead of docker, and private ORG):
# VERSION=7.6.0-6 ORG=vmogilev_rhmi BUILD_FRESH=true BUILD_TOOL=podman make create/rhsso-operator/index



REG="${REG:-quay.io}"
ORG="${ORG:-integreatly}"
IMAGE="${IMAGE:-sso7-rhel8-operator-index}"
BUILD_TOOL="${BUILD_TOOL:-docker}"
BUILD_FRESH="${BUILD_FRESH:-true}"
VERSION="${VERSION}"



generate_from() {
    if [ -z $1 ]; then
        echo "generate_from() called without a parameter"
        exit 1
    fi
    bundle=$(yq e ".bundles[-1] | .name |= sub(\"rhsso-operator.\",\"\") \
        | select(.name == \"$VERSION\") | .image" \
        bundles/rhsso-operator/bundles.yaml)
    
    if [[ -z $bundle ]]; then
        echo "No matching bundle, exiting"
        exit 1
    fi

    index=$(yq e ".products[\"rhsso\"].index" products/installation.yaml)

    printf "Building index $1\n\tFrom: $index\n\tIncluding: $bundle\n"

    echo "Building index $1"
    opm index add \
      --enable-alpha \
      --bundles $bundle \
      --from-index $index \
      --container-tool $BUILD_TOOL \
      --tag $1
}

generate_full() {
    if [ -z $1 ]; then
        echo "generate_full() called without a parameter"
        exit 1
    fi

    bundles=$(yq e ".bundles[] | .name |= sub(\"rhsso-operator.\",\"\")  | .image" \
        bundles/rhsso-operator/bundles.yaml)


    if [[ -z $bundles ]]; then
        echo "No matching bundles, exiting"
        exit 1
    fi

    printf "Including bundles:\n$bundles\n\n"

    echo "Building index $1"

    delim=""
    bundle_csv=""
    for item in $bundles; do
        bundle_csv="$bundle_csv$delim$item"
        delim=","
    done

    opm index add \
      --enable-alpha \
      --bundles $bundle_csv \
      --container-tool $BUILD_TOOL \
      --tag $1
}


generate_index() {
    if [[ $VERSION =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$ ]]; then
        echo "Valid version string: ${VERSION}"
    else
        echo "Error: Invalid version string: ${VERSION}"
        exit 1
    fi

    echo "Building index for v$VERSION"

    tag=$(echo $VERSION | sed 's/.p$//g' | sed 's/+/-/g')
    image="$REG/$ORG/$IMAGE:$tag"

    if [ "$BUILD_FRESH" = true ]; then
        generate_full $image
    else
        generate_from $image
    fi
}

generate_index
