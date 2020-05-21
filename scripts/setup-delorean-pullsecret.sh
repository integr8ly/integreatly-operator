#!/usr/bin/env bash
set -e
set -o pipefail

CONFIG_PULLSECRET="$(pwd)/config-pullsecret"
DECODED_PULLSECRET="$(pwd)/config-pullsecret-decoded"
DELOREAN_PULLSECRET="$(pwd)/integreatly-delorean-auth.json"
STAGED_PULLSECRET="$(pwd)/staged-pullsecret"
COMBINED_PULLSECRET="$(pwd)/combined-pullsecret"

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

sed -i 's+quay.io+quay.io/integreatly/delorean+g' $DELOREAN_PULLSECRET

oc get secret pull-secret -n openshift-config -o yaml > "$CONFIG_PULLSECRET"
yq r $CONFIG_PULLSECRET 'data' | awk '{print $2}' | base64 -d > $DECODED_PULLSECRET
jq -s -c '{auths: map(.auths) | add}' $DECODED_PULLSECRET $DELOREAN_PULLSECRET | base64 > $STAGED_PULLSECRET
awk '{ printf "%s", $0 }' $STAGED_PULLSECRET  > $COMBINED_PULLSECRET
oc patch secret pull-secret -n openshift-config -p='{"data": {".dockerconfigjson": "'$(cat ${COMBINED_PULLSECRET})'"}}'
rm $CONFIG_PULLSECRET $STAGED_PULLSECRET $COMBINED_PULLSECRET $DECODED_PULLSECRET

echo "waiting 10 minutes to allow cluster to stabilize ..."
sleep 10m

echo "secret 'pull-secret' patched in namespace 'openshift-config' to add delorean quay.io access!"
