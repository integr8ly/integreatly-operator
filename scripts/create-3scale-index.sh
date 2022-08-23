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
# BUILD_FRESH - if set to true we build an index from scratch, otherwise we append to the existing index
#    in products/installation.yaml
# VERSION - index version to build: 
#   BUILD_FRESH=false; this defines the bundle version to append to the existing index
#   BUILD_FRESH=true ; this defines the max version and includes all bundles beneath it
#
# Example:
# VERSION=0.8.3+0.1645735250.p ORG=acatterm make create/3scale/index


REG="${REG:-quay.io}"
ORG="${ORG:-integreatly}"
IMAGE="${IMAGE:-3scale-index}"
BUILD_TOOL="${BUILD_TOOL:-docker}"
BUILD_FRESH="${BUILD_FRESH:-false}"
VERSION="${VERSION}"



generate_from() {
    if [ -z $1 ]; then
        echo "generate_from() called without a parameter"
        exit 1
    fi
    bundle=$(yq e ".bundles[] | .name |= sub(\"3scale-operator.\",\"\") \
        | select(.name == \"v$VERSION\") | .image" \
        bundles/3scale-operator/bundles.yaml)
    
    if [[ -z $bundle ]]; then
        echo "No matching bundle, exiting"
        exit 1
    fi

    index=$(yq e ".products[\"3scale\"].index" products/installation.yaml)

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

    bundles=$(yq e ".bundles[] | .name |= sub(\"3scale-operator.\",\"\") \
        | select(.name <= \"v$VERSION\") | .image" \
        bundles/3scale-operator/bundles.yaml)


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
    image="$REG/$ORG/$IMAGE:v$tag"

    if [ "$BUILD_FRESH" = true ]; then
        generate_full $image
    else
        generate_from $image
    fi
}

generate_index
