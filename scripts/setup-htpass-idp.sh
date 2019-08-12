#!/usr/bin/env bash

PASSWORD=$(openssl rand -base64 12)

echo Setting up htpasswd IDP

if [[ ! -f "htpasswd" ]]; then
  echo creating htpasswd file
  touch htpasswd
fi

htpasswd -b htpasswd customer-admin ${PASSWORD}
htpasswd -b htpasswd cluster-admin ${PASSWORD}

echo user added customer-admin ${PASSWORD}
oc delete secret htpasswd-secret -n openshift-config
oc create secret generic htpasswd-secret --from-file=htpasswd=htpasswd -n openshift-config
oc patch oauth cluster --type=merge -p '{ "spec": { "identityProviders": [{ "name": "Test Identity Provider", "challenge": true, "login": true, "mappingMethod": "claim", "type": "HTPasswd", "htpasswd": { "fileData": { "name": "htpasswd-secret" } } }] } }'