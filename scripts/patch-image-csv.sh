#!/bin/bash
#
# About: Script to patch the csv created during addon flow with dev image
#
# Usage:
# VERSION=<your image tag>
# ORG=<you quay org>
# ADDON_VERSION=<version set https://gitlab.cee.redhat.com/service/managed-tenants/-/blob/main/addons/managed-api-service/metadata/stage/addon.yaml#L25>
# Set the varse.g.
# export VERSION=v1.6.0
# export ADDON_VERSION=v1.6.0
# export ORG=austincunningham
# ./patch-image-csv.sh
oc patch ClusterServiceVersion managed-api-service.$ADDON_VERSION -n redhat-rhoam-operator --patch '{"metadata": {"annotations": {"containerImage": "quay.io/'$ORG'/managed-api-service:'$VERSION'" }}}' --type=merge
oc patch ClusterServiceVersion managed-api-service.$ADDON_VERSION -n redhat-rhoam-operator --type='json' -p='[{"op": "replace", "path": "/spec/install/spec/deployments/0/spec/template/spec/containers/0/image", "value":"quay.io/'$ORG'/managed-api-service:'$VERSION'"}]'
echo Verificaton the CSV has been patched :
oc get csv managed-api-service.$VERSION -n redhat-rhoam-operator -o yaml | grep "$ORG"
echo " "
echo You will have to patch the RHMI CR when its created with useClusterStorge e.g.
echo oc patch rhmi rhoam -n redhat-rhoam-operator --patch \'{ \"spec\": {\"useClusterStorage\": \"true\" }}\' --type=merge
