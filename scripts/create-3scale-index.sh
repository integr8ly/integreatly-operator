#!/usr/bin/env bash
#
# Prereq:
# - opm
# - yq (4.24.2+)
#
# Function:
# Script creates 3scale index/indices for RHOAM 
#
# Usage:
# REG=<REGISTRY> ORG=<QUAY ORG> IMAGE=<IMAGE NAME> BUILD_TOOL=<docker|podman>  VERSION=<VERSION TAG> make create/3scale/index
# REG - registry of where to push the bundles and indices, defaults to quay.io
# ORG - organization of where to push the bundles and indices
# IMAGE - image name of the image to push
# BUILD_TOOL - tool used for building the bundle and index, defaults to docker
# VERSION - index version to build; this defines the max version and includes all bundles beneath it
# TAG - version tag to be used for the image
#
# Example:
# VERSION=0.8.3+0.1645735250.p ORG=acatterm make create/3scale/index


REG="${REG:-quay.io}"
ORG="${ORG:-integreatly}"
IMAGE="${IMAGE:-3scale-index}"
BUILD_TOOL="${BUILD_TOOL:-docker}"
VERSION="${VERSION}"


generate_index() {

    if [[ $VERSION =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*)?(\+[0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*)?$ ]]; then
        echo "Valid version string: ${VERSION}"
    else
        echo "Error: Invalid version string: ${VERSION}"
        exit 1
    fi

    echo "Building index for v$VERSION"

    bundles=$(yq e ".bundles[] | .name |= sub(\"3scale-operator.\",\"\") \
        | select(.name <= \"v$VERSION\") | .image" \
        bundles/3scale-operator/bundles.yaml)


    if [[ -z $bundles ]]; then
        echo "No matching bundles, exiting"
        exit 1
    fi

    printf "Including bundles:\n$bundles\n"

    delim=""
    bundle_csv=""
    for item in $bundles; do
        bundle_csv="$bundle_csv$delim$item"
        delim=","
    done

    tag=$(echo $VERSION | sed 's/.p$//g' | sed 's/+/-/g')
    image="$REG/$ORG/$IMAGE:v$tag"

    echo "Building index $image"
    opm index add \
      --enable-alpha \
      --bundles $bundle_csv \
      --container-tool $BUILD_TOOL \
      --tag $image
}

generate_index
