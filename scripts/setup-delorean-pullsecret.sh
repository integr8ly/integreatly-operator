#!/usr/bin/env bash
set -o pipefail

CONFIG_PULLSECRET="$(pwd)/config-pullsecret"
DECODED_PULLSECRET="$(pwd)/config-pullsecret-decoded"
DELOREAN_PULLSECRET="$(pwd)/integreatly-delorean-auth.json"
STAGED_PULLSECRET="$(pwd)/staged-pullsecret"
COMBINED_PULLSECRET="$(pwd)/combined-pullsecret"
TEMP_SERVICEACCOUNT_NAME="rhmi-operator"

if [ "$INSTALLATION_TYPE" = "managed-api" ]; then
  NAMESPACE_PREFIX="${NAMESPACE_PREFIX:-redhat-rhoam-}"
fi
NAMESPACE_PREFIX="${NAMESPACE_PREFIX:-redhat-rhmi-}"
OPERATOR_NAMESPACE="${NAMESPACE_PREFIX}operator"

if [ ! -z "${DELOREAN_DOCKER_CONFIG}" ]; then
    echo -e $DELOREAN_DOCKER_CONFIG > $DELOREAN_PULLSECRET
fi

if [ ! -f ${DELOREAN_PULLSECRET} ]; then
  echo "Error: ${DELOREAN_PULLSECRET} does not exist!"
  echo "A valid docker config to access the delorean quay.io registry is required"
  echo "Download the integreatly-delorean-auth.json file from https://quay.io/organization/integreatly?tab=robots Docker Configuration tab"
  echo "Please contact the delorean team if you need further help!!"
  exit 1
fi

#Assumes the pull secret file contains only one auth entry and it's the one we want
DELOREAN_PASSWORD="$(jq -r '.auths|to_entries[0].value.auth' $DELOREAN_PULLSECRET | base64 --decode | awk -F : '{print $2}')"
DELOREAN_USERNAME="$(jq -r '.auths|to_entries[0].value.auth' $DELOREAN_PULLSECRET | base64 --decode | awk -F : '{print $1}')"

oc get secret pull-secret -n openshift-config -o yaml > "$CONFIG_PULLSECRET"
yq r $CONFIG_PULLSECRET 'data' | awk '{print $2}' | base64 -d > $DECODED_PULLSECRET
combine_and_deploy_cluster_secret() {
    jq -s -c '{auths: map(.auths) | add}' $DECODED_PULLSECRET $DELOREAN_PULLSECRET | base64 > $STAGED_PULLSECRET
    awk '{ printf "%s", $0 }' $STAGED_PULLSECRET  > $COMBINED_PULLSECRET
    oc patch secret pull-secret -n openshift-config -p='{"data": {".dockerconfigjson": "'$(cat ${COMBINED_PULLSECRET})'"}}'
    rm -f $CONFIG_PULLSECRET $STAGED_PULLSECRET $COMBINED_PULLSECRET $DECODED_PULLSECRET
    echo "waiting 10 minutes to allow cluster to stabilize ..."
    sleep 10m
    echo "secret 'pull-secret' patched in namespace 'openshift-config' to add delorean quay.io access!"
}

setup_ns_and_local_secret() {
  oc new-project ${NAMESPACE_PREFIX}3scale --as system:serviceaccount:${OPERATOR_NAMESPACE}:${TEMP_SERVICEACCOUNT_NAME}
  oc create secret docker-registry --docker-server=quay.io --docker-username="${DELOREAN_USERNAME}" --docker-password="${DELOREAN_PASSWORD}" regsecret -n ${NAMESPACE_PREFIX}3scale --as system:serviceaccount:${OPERATOR_NAMESPACE}:${TEMP_SERVICEACCOUNT_NAME}
  oc secrets link default regsecret --for=pull -n ${NAMESPACE_PREFIX}3scale --as system:serviceaccount:${OPERATOR_NAMESPACE}:${TEMP_SERVICEACCOUNT_NAME}

  if [ "$INSTALLATION_TYPE" = "managed" ]; then
    oc new-project ${NAMESPACE_PREFIX}fuse --as system:serviceaccount:${OPERATOR_NAMESPACE}:${TEMP_SERVICEACCOUNT_NAME}
    oc create secret docker-registry --docker-server=quay.io --docker-username="${DELOREAN_USERNAME}" --docker-password="${DELOREAN_PASSWORD}" regsecret -n ${NAMESPACE_PREFIX}fuse --as system:serviceaccount:${OPERATOR_NAMESPACE}:${TEMP_SERVICEACCOUNT_NAME}
    oc secrets link default regsecret --for=pull -n ${NAMESPACE_PREFIX}fuse --as system:serviceaccount:${OPERATOR_NAMESPACE}:${TEMP_SERVICEACCOUNT_NAME}
  fi
  
  oc project ${OPERATOR_NAMESPACE}
}

if [ -n "$(cat ${DECODED_PULLSECRET} | grep delorean)" ]; then
  echo "Delorean secret found on cluster"
  rm $CONFIG_PULLSECRET $DECODED_PULLSECRET
else
  if [ -n "$(cat ${DELOREAN_PULLSECRET} | grep delorean)" ]; then
    combine_and_deploy_cluster_secret
  else
    sed -i 's+quay.io+quay.io/integreatly/delorean+g' $DELOREAN_PULLSECRET
    combine_and_deploy_cluster_secret
  fi
fi

echo "Setting up secrets for imagestreams"
setup_ns_and_local_secret
