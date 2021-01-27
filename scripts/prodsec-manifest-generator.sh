#!/usr/bin/env bash

# This script generates a manifest of the integreatly-operator which includes the direct and indirect dependencies 
# used. The generated manifest corresponds to a released version and is located in : ../rhoam-manifests/

# Compares current manifest vs manifest generated for master
manifest_compare() {
    # Additional sorting of the master manifest is required as it seems that prow sorts files in a different manner.
    sort -u "$RHOAM_CURRENT_MASTER" > "$RHOAM_SORTED_MASTER"

    MANIFESTS_DIFF=$(diff --suppress-common-lines ${RHOAM_MASTER_FROM_BRANCH} ${RHOAM_SORTED_MASTER})
    if [ ! -z "$MANIFESTS_DIFF" ]; then
        echo "Difference found between master manifests, run `TYPE_OF_MANIFEST=master make manifest/prodsec` and push PR again"
        
        # Delete sorted files
        rm -f "$RHOAM_MASTER_FROM_BRANCH"
        rm -f "$RHOAM_SORTED_MASTER"
        exit 1
    else
        echo "No difference found between the manifests"

        # Delete sorted files
        rm -f "$RHOAM_MASTER_FROM_BRANCH"
        rm -f "$RHOAM_SORTED_MASTER"
        exit 0
    fi
}

# Generates the manifest, it can be either master or production manifest
manifest_generate() {
    # Pre-filetered manifest file
    PRE_SORTED_FILE="prodsec-manifests/pre-sorted-file.txt"

    # Dependencies used
    go mod graph | cut -d " " -f 2 | tr @ - | while read x; do echo "${SERVICE_NAME}:${VERSION}/$x" >> "$PRE_SORTED_FILE"; done

    # Remove repeating dependencies
    sort -u "$PRE_SORTED_FILE" > "prodsec-manifests/$FILE_NAME"

    # Delete pre-sorted file
    rm -f "$PRE_SORTED_FILE"
}



case $TYPE_OF_MANIFEST in
"master")
    SERVICE_NAME="services-rhoam"
    VERSION="master"
    FILE_NAME="rhoam-master-manifest.txt"

    manifest_generate
    exit 0
    ;;
"compare")
    SERVICE_NAME="services-rhoam"
    VERSION="master"
    FILE_NAME="master-from-branch-manifest.txt"

    RHOAM_MASTER_FROM_BRANCH="prodsec-manifests/master-from-branch-manifest.txt"
    RHOAM_CURRENT_MASTER="prodsec-manifests/rhoam-master-manifest.txt"
    RHOAM_SORTED_MASTER="prodsec-manifests/rhoam-master-sorted-manifest.txt"

    manifest_generate
    manifest_compare
    exit 0
    ;;
"production")
    case $OLM_TYPE in
      "integreatly-operator")
        OLM_TYPE="rhmi"
        VERSION=$(grep integreatly-operator deploy/olm-catalog/integreatly-operator/integreatly-operator.package.yaml | awk -F v '{print $2}')
        FILE_NAME="rhmi-production-release-manifest.txt"
        SERVICE_NAME="services-rhmi"
        ;;
      "managed-api-service")
        OLM_TYPE="rhoam"
        VERSION=$(grep managed-api-service deploy/olm-catalog/managed-api-service/managed-api-service.package.yaml | awk -F v '{print $3}')
        FILE_NAME="rhoam-production-release-manifest.txt"
        SERVICE_NAME="services-rhoam"
        ;;
      *)
        echo "Invalid OLM_TYPE set"
        echo "Use \"integreatly-operator\" or \"managed-api-service\""
        exit 1
        ;;
    esac
    manifest_generate
    ;;
*)
    echo "Invalid type of manifest requested"
    echo "Use \"master\",\"production\" or \"compare\""
    exit 1
    ;;
esac
