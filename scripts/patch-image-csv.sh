#!/bin/bash
#
# About: Script to patch the csv created during addon flow with dev image and patch the rhmi cr with useClusterStorage
#
# Usage:
# VERSION=<your image tag>
# ORG=<your quay org>
# ADDON_VERSION=<version set https://gitlab.cee.redhat.com/service/managed-tenants/-/blob/main/addons/rhoams/metadata/stage/addon.yaml#L25>
# USE_CLUSTER_STORAGE=<true/false optional will default true if not set>
# Set the vars.g.
# export VERSION=v1.6.0
# export ADDON_VERSION=v1.6.0
# export ORG=austincunningham
# export USE_CLUSTER_STORAGE=false
# ./patch-image-csv.sh
#
# Check csv exists
oc get csv managed-api-service.$ADDON_VERSION -n redhat-rhoam-operator > /dev/null 2>&1
while [ $? -ne 0 ]; do
  oc get csv managed-api-service.$ADDON_VERSION -n redhat-rhoam-operator > /dev/null 2>&1
done
# Patch the csv
oc patch ClusterServiceVersion managed-api-service.$ADDON_VERSION -n redhat-rhoam-operator --patch '{"metadata": {"annotations": {"containerImage": "quay.io/'$ORG'/managed-api-service:'$VERSION'" }}}' --type=merge
oc patch ClusterServiceVersion managed-api-service.$ADDON_VERSION -n redhat-rhoam-operator --type='json' -p='[{"op": "replace", "path": "/spec/install/spec/deployments/0/spec/template/spec/containers/0/image", "value":"quay.io/'$ORG'/managed-api-service:'$VERSION'"}]'
echo Verificaton the CSV has been patched :
oc get csv managed-api-service.$VERSION -n redhat-rhoam-operator -o yaml | grep "$ORG"
echo " "
echo Patching the RHMI CR when its created with useClusterStorage
# Check the rhmi cr exists
oc get rhmi rhoam -n redhat-rhoam-operator > /dev/null 2>&1
while [ $? -ne 0 ]; do
  oc get rhmi rhoam -n redhat-rhoam-operator > /dev/null 2>&1
done
# default USE_CLUSTER_STORAGE to true if not set
if [ -z ${USE_CLUSTER_STORAGE+x} ]; then
  export USE_CLUSTER_STORAGE=true
fi
# Patch rhmi CR
oc patch rhmi rhoam -n redhat-rhoam-operator --patch '{ "spec": {"useClusterStorage": "'$USE_CLUSTER_STORAGE'" }}' --type=merge
echo Verification USE_CLUSTER_STORAGE is set :
oc get rhmi rhoam -n redhat-rhoam-operator -o yaml | grep useClusterStorage
