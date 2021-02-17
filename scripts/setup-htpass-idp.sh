#!/usr/bin/env bash

PASSWORD="${CUSTOM_PWD:-$(openssl rand -base64 12)}"

echo Setting up htpasswd IDP

oc get secret htpasswd-secret -n openshift-config -o 'go-template={{index .data "htpasswd"}}' | base64 --decode > htpasswd

if [[ ! -f "htpasswd" ]]; then
  echo creating htpasswd file
  touch htpasswd
fi

htpasswd -b htpasswd customer-admin "${PASSWORD}"
echo user added customer-admin "${PASSWORD}"

htpasswd -b htpasswd test-user "${PASSWORD}"
echo user added test-user "${PASSWORD}"

htpasswd -b htpasswd sre-user-1 "${PASSWORD}"
echo user added sre-user-1 "${PASSWORD}"

oc delete secret htpasswd-secret -n openshift-config
oc create secret generic htpasswd-secret --from-file=htpasswd=htpasswd -n openshift-config
oc patch oauth cluster --type=merge -p '{ "spec": { "identityProviders": [{ "name": "htpasswd_provider", "challenge": true, "login": true, "mappingMethod": "claim", "type": "HTPasswd", "htpasswd": { "fileData": { "name": "htpasswd-secret" } } }] } }'

oc adm groups add-users dedicated-admins customer-admin
oc adm groups add-users osd-sre-admins sre-user-1

# Also configure session expiration (OAuth access token duration) and token inactivity timeouts if both of
# TOKEN_INACTIVITY_TIMEOUT and SESSION_EXPIRATION_TIMEOUT environment variables were specified
# See: https://github.com/integr8ly/integreatly-operator/pull/1561#discussion_r566491866 for examples
# See also script help for list of supported environment variables and additional examples
if [ "x${TOKEN_INACTIVITY_TIMEOUT}" != x ] && [ "x${SESSION_EXPIRATION_TIMEOUT}" != x ]
then
  # shellcheck source=/dev/null
  source "$(dirname "$0")/refine-oauth-token-timeouts.sh"
fi
