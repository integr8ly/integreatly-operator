#!/bin/sh
# Prereq:
# - oc
# - jq
# Usage:
# ./podsAz.sh [<namespace>]
# <namespace> parameter is optional, defaults to ALL namespaces when ommited
# Function:
# Lists the Pods for a namespace and their Availability Zones

NODES_ARR=()
ZONES_ARR=()

# get list of node names and their zones and push this information into two arrays
nodes=$(oc get nodes -o json | jq -r '.items[].metadata | .name + " " + .labels["topology.kubernetes.io/zone"]')
while read -a line; do
  NODES_ARR+=(${line[0]})
  ZONES_ARR+=(${line[1]})
done <<< "$nodes"

# returns associated zone for a node name passed as parameter
get_zone() {
  for i in ${!NODES_ARR[@]}; do
    if [[ ${NODES_ARR[$i]} == $1 ]]; then
      echo ${ZONES_ARR[$i]}
      break
    fi
  done
}

NAMESPACE=$1
if [[ $NAMESPACE == "" ]]; then
  echo "Pods distribution for ALL namespaces"
  NAMESPACE_ARG="--all-namespaces"
else
  echo "Pods distribution for '$NAMESPACE'"
  NAMESPACE_ARG="-n $NAMESPACE"
fi

pods=$(oc get pods $NAMESPACE_ARG -o json | jq -r '.items[] | .metadata.namespace + " " + .metadata.name + " " + .spec.nodeName')
echo "| Pod namespace | Pod name | Availability Zone |"
echo "| ------------- | -------- | ----------------- |"
while read -a pod; do
  echo "| ${pod[0]} | ${pod[1]} | $(get_zone ${pod[2]}) |"
done <<< "$pods"