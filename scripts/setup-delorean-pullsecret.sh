#!/usr/bin/env bash
set -e
set -o pipefail

CONFIG_PULLSECRET="$(pwd)/config-pullsecret"
DECODED_PULLSECRET="$(pwd)/config-pullsecret-decoded"
DELOREAN_PULLSECRET="$(pwd)/integreatly-delorean-auth.json"
STAGED_PULLSECRET="$(pwd)/staged-pullsecret"
COMBINED_PULLSECRET="$(pwd)/combined-pullsecret"

if [ ! -f ${DELOREAN_PULLSECRET} ]; then
  echo "Error: integreatly-delorean-auth.json not found at ${DELOREAN_PULLSECRET}"
  echo "Download the integreatly-delorean-auth.json file from https://quay.io/organization/integreatly?tab=robots Docker Configuration tab"
  exit 1
fi

setup_global() {
    oc get secret pull-secret -n openshift-config -o yaml > "$CONFIG_PULLSECRET"
    sed -i 's+quay.io+quay.io/integreatly/delorean+g' $DELOREAN_PULLSECRET
    yq r $CONFIG_PULLSECRET 'data' | awk '{print $2}' | base64 -d > $DECODED_PULLSECRET
    jq -s -c '{auths: map(.auths) | add}' $DECODED_PULLSECRET $DELOREAN_PULLSECRET | base64 > $STAGED_PULLSECRET
    awk '{ printf "%s", $0 }' $STAGED_PULLSECRET  > $COMBINED_PULLSECRET
    oc patch secret pull-secret -n openshift-config -p='{"data": {".dockerconfigjson": "'$(cat ${COMBINED_PULLSECRET})'"}}'
    rm $CONFIG_PULLSECRET $STAGED_PULLSECRET $COMBINED_PULLSECRET $DECODED_PULLSECRET
}

setup_local() {
    echo -e $DELOREAN_DOCKERCONFIG_SECRET_YAML | oc apply --namespace=$NAMESPACE -f -
}

setup_local