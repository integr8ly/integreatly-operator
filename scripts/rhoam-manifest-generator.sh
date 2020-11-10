#!/usr/bin/env bash

# This script generates a manifest of the integreatly-operator which includes the direct and indirect dependencies 
# used. The generated manifest corresponds to a released version and is located in : ../rhoam-manifests/

# Set the service name and version
SERVICE_NAME="services-rhoam"

# Pull rhoam version
RHOAM_VERSION=$(grep managed-api-service.v deploy/olm-catalog/managed-api-service/managed-api-service.package.yaml | tail -c 6)

# Pre-filetered manifest file
pre_sorted_file="pre-sorted-file.txt"

# Output file name
file="${RHOAM_VERSION}-manifest.txt"

# Dependencies used
go mod graph | cut -d " " -f 2 | tr @ - | while read x; do echo "${SERVICE_NAME}:${RHOAM_VERSION}/$x" >> "rhoam-manifests/$pre_sorted_file"; done

# Remove repeating dependencies
sort -u "rhoam-manifests/$pre_sorted_file" > "rhoam-manifests/$file"

# Delete pre-sorted file
rm -f "rhoam-manifests/$pre_sorted_file"
