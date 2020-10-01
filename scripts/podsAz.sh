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
zone=`oc get nodes $node -o json | jq -r '.metadata.labels["topology.kubernetes.io/zone"]'`
echo "| $pod_name | $zone |"
done <<< "$pods"