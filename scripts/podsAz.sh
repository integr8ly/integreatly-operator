#!/bin/sh
# Prereq:
# - oc
# - jq
# Usage:
# ./podsAz.sh <namespace>
# Function:
# Lists the Pods for an namespace and there Availability Zones

NAMESPACE=$1
echo "Pods distribution for '$NAMESPACE'"
pods=`oc get pods -n $NAMESPACE -o json | jq -r '.items[].metadata.name'`
echo "| Pod name | Availability Zone |"
echo "| -------- | ----------------- |"
while IFS= read -r pod_name; do
node=`oc get pods/$pod_name -n $NAMESPACE -o json | jq -r '.spec.nodeName'`
machine_name=`oc get nodes $node -o json | jq -r '.metadata.annotations["machine.openshift.io/machine"] | match("openshift-machine-api\/(.+)") | .captures[0].string'`
zone=`oc get machines $machine_name -n openshift-machine-api -o json | jq -r '.metadata.labels["machine.openshift.io/zone"]'`
echo "| $pod_name | $zone |"
done <<< "$pods"