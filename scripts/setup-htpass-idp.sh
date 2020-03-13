#!/usr/bin/env bash
if [ "$CUSTOM_PWD" ]; then PASSWORD="$CUSTOM_PWD"; else PASSWORD=$(openssl rand -base64 12); fi

echo Setting up htpasswd IDP

oc get secret htpasswd-secret -n openshift-config -o 'go-template={{index .data "htpasswd"}}' | base64 --decode > htpasswd

if [[ ! -f "htpasswd" ]]; then
  echo creating htpasswd file
  touch htpasswd
fi

htpasswd -b htpasswd customer-admin ${PASSWORD}
echo user added customer-admin ${PASSWORD}

htpasswd -b htpasswd test-user ${PASSWORD}
echo user added test-user ${PASSWORD}

htpasswd -b htpasswd sre-user-1 ${PASSWORD}
echo user added sre-user-1 ${PASSWORD}

oc delete secret htpasswd-secret -n openshift-config
oc create secret generic htpasswd-secret --from-file=htpasswd=htpasswd -n openshift-config
oc patch oauth cluster --type=merge -p '{ "spec": { "identityProviders": [{ "name": "htpasswd_provider", "challenge": true, "login": true, "mappingMethod": "claim", "type": "HTPasswd", "htpasswd": { "fileData": { "name": "htpasswd-secret" } } }] } }'

oc adm groups add-users dedicated-admins customer-admin
oc adm groups add-users osd-sre-admins sre-user-1
#uMlQslDI4V0Mpgnc