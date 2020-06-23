#!/usr/bin/env bash

timestamp() {
  date +%d%m%Y
}

timestamp

SERVICE_NAME=rhmi
COMPONENT_NAME=integreatly-operator

file="rh-manifest-$(timestamp).txt"
touch $file

MW_MANIFESTS=( 3scale amq-online apicurito codeready-workspaces fuse-online)
RHMI_MANIFESTS=( cloud-resources rhsso solution-explorer unifiedpush)

echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/Dockerfile-FROM registry.access.redhat.com/ubi8/ubi" > $file

for m in "${MW_MANIFESTS[@]}"
do
  echo "${m}"
  csv=$(yq r manifests/integreatly-${m}/${m}.package.yaml channels[0].currentCSV)
  echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/OLM-BUNDLE-$csv" >> $file
done

#Include direct dependencies only
go mod graph | grep ${COMPONENT_NAME} | cut -d " " -f 2 | while read x; do echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/$x" >> $file; done

#for m in "${RHMI_MANIFESTS[@]}"
#do
#  echo "${m}"
#  csv=$(yq r manifests/integreatly-${m}/${m}.package.yaml channels[0].currentCSV)
#  echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/OLM-BUNDLE-$csv" >> $file
#done

COMPONENT_NAME=cloud-resource-operator
echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/Dockerfile-FROM registry.access.redhat.com/ubi8/ubi" >> $file
(cd /home/mnairn/go/src/github.com/integr8ly/cloud-resource-operator && go mod graph) | grep ${COMPONENT_NAME}| cut -d " " -f 2 | while read x; do echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/$x" >> $file; done

#Include all dependencies
#go list -m all | while read x; do echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/$x" >> $file; done

#Include direct dependencies only
#go mod graph | grep ${COMPONENT_NAME} | cut -d " " -f 2 | while read x; do echo "mgmt_services/${SERVICE_NAME}:${COMPONENT_NAME}/$x" >> $file; done
