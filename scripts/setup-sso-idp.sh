#!/usr/bin/env bash
set -e
set -o pipefail 

dedicated_admins=( "customer-admin01" "customer-admin02" "customer-admin03" )

PASSWORD="${PASSWORD:-$(openssl rand -base64 12)}"
REALM="${REALM:-testing-idp}"
INSTALLATION_PREFIX="${INSTALLATION_PREFIX:-$(oc get RHMIs --all-namespaces -o json | jq -r .items[0].spec.namespacePrefix)}"
INSTALLATION_PREFIX=${INSTALLATION_PREFIX%-} # remove trailing dash

echo "User password set to \"${PASSWORD}\""

CLIENT_SECRET=$(openssl rand -base64 20)
OAUTH_URL=https://$(oc get route oauth-openshift -n openshift-authentication -o json | jq -r .spec.host)
KEYCLOAK_URL=https://$(oc get route keycloak-edge -n $INSTALLATION_PREFIX-rhsso -o json | jq -r .spec.host)
echo "Keycloak console: $KEYCLOAK_URL/auth/admin/master/console/#/realms/$REALM"
echo "Keycloack credentials: admin / $(oc get secret credential-rhsso -n $INSTALLATION_PREFIX-rhsso -o json | jq -r .data.ADMIN_PASSWORD | base64 --decode)"
echo "Keycloak realm: $REALM"

# If CLUSTER_ID is not passed, find out ID based on currently targeted server
set +e # ignore errors in environments without ocm command
CLUSTER_ID="${CLUSTER_ID:-$(ocm get /api/clusters_mgmt/v1/clusters/ 2>/dev/null | jq -r ".items[] | select(.api.url == \"$(oc cluster-info | grep -Eo 'https?://[-a-zA-Z0-9\.:]*')\") | .id ")}"
set -e

if [[ ${CLUSTER_ID} ]]; then
  # If CLUSETER_ID is detected - use OCM for IDP management
  echo "Cluster ID: $CLUSTER_ID"

  IDP_ID=$(ocm get "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers" | jq -r "select(.size > 0) | .items[] | select( .name == \"$REALM\") | .id")
  if [[ ${IDP_ID} ]]; then
    echo "$REALM IDP is already present in OCM configuration."
    echo "OpenShift resources from testing-idp-template.yml will not be applied"
    echo "If you would like to re-apply any resources, delete the IDP from OCM and re-run this script."
    echo "To delete IDP execute: ocm delete \"/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers/$IDP_ID\""
  else

    oc process -p OAUTH_URL=$OAUTH_URL -p NAMESPACE=$INSTALLATION_PREFIX-rhsso -p REALM=$REALM -p PASSWORD=$PASSWORD -p CLIENT_SECRET=$CLIENT_SECRET -f ${BASH_SOURCE%/*}/testing-idp-template.yml | oc apply -f -

    sed "s|REALM|$REALM|g; s|KEYCLOAK_URL|$KEYCLOAK_URL|g; s|CLIENT_SECRET|$CLIENT_SECRET|g" "${BASH_SOURCE%/*}/ocm-idp-template.json" | ocm post /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers
    echo "$REALM IDP added into OCM configuration"
  fi

  echo "Adding users into dedicated-admins group:"
  for username in "${dedicated_admins[@]}"
  do
    if [[ $(ocm get "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/groups/dedicated-admins/users" | jq ".items[] | select( .id == \"$username\")") ]]; then
      echo "$username is already in dedicated-admins group" 
    else
      echo '{"id":"'$username'"}' | ocm post /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/groups/dedicated-admins/users
      echo "$username added to dedicated-admins group"
    fi
  done

  until oc get oauth cluster -o json | jq .spec.identityProviders[].name | grep -q $REALM; do echo "\"cluster\" OAuth configuration does not contain our IDP yet, trying again in 10s"; sleep 10s; done


else
  # If CLUSETER_ID is not detected - manage IDP directly in cluster config
  echo "No CLUSTER_ID detected. IDP will be added directly into \"cluster\" OAuth resource."

  if [[ ! $(oc get group dedicated-admins) ]]; then
    # create dedicated-admins group if it doesn't exist
    echo '{"kind": "Group", "apiVersion": "user.openshift.io/v1", "metadata": { "name": "dedicated-admins" }, "users": null }' | oc create -f -
  fi

  # apply KeycloakRealm, KeycloakClient and KeylcloakUser(s) resources from a template
  oc process -p OAUTH_URL=$OAUTH_URL -p NAMESPACE=$INSTALLATION_PREFIX-rhsso -p REALM=$REALM -p PASSWORD=$PASSWORD -p CLIENT_SECRET=$CLIENT_SECRET -f ${BASH_SOURCE%/*}/testing-idp-template.yml | oc apply -f -

  # update the Secret used by OpenShift OAuth server
  CLIENT_SECRET_NAME="idp-$REALM"
  oc delete secret $CLIENT_SECRET_NAME -n openshift-config 2>/dev/null || true
  oc create secret generic $CLIENT_SECRET_NAME --from-literal=clientSecret=$CLIENT_SECRET -n openshift-config

  CA_CONFIGMAP_NAME=""
  if [[ $(curl -sSo /dev/null $KEYCLOAK_URL 2>&1 | grep "self signed certificate") ]]; then
    # update the Secret that stores CA cert
    CA_CONFIGMAP_NAME="idp-ca-$REALM"
    oc get secret router-ca -o yaml -n openshift-ingress-operator -o 'go-template={{index .data "tls.crt"}}' | base64 --decode > router-ca.tmp
    oc delete configmap $CA_CONFIGMAP_NAME -n openshift-config 2>/dev/null || true
    oc create configmap $CA_CONFIGMAP_NAME --from-file=ca.crt=router-ca.tmp -n openshift-config
  fi

  if [[ ! $(oc get oauth cluster -o yaml | grep identityProviders) ]]; then
    # add an empty array into .spec.identityProviders of "cluster" OAuth resource if it was set to null
    oc patch oauth cluster --type=json -p '[{"op":"add", "path":"/spec/identityProviders", "value":[]}]'
  fi
  # try to add the testing IDP .spec.identityProviders of "cluster" OAuth resource
  if [[ $(oc patch oauth cluster --type=json -p '[{"op":"add", "path":"/spec/identityProviders/-", "value":{ "name": "'$REALM'", "mappingMethod": "claim", "type": "OpenID", "openID": { "ca": {"name": "'$CA_CONFIGMAP_NAME'"}, "clientID": "openshift", "clientSecret": { "name": "'$CLIENT_SECRET_NAME'" }, "issuer": "'$KEYCLOAK_URL'/auth/realms/'$REALM'", "claims": { "preferredUsername": ["preferred_username"], "email": ["email"], "name": ["name"] } } } }]' 2>&1) =~ "must have a unique name" ]]; then
    echo "$REALM IDP already exists in \"cluster\" OAuth, this resource will not be updated!"
  fi

  for username in "${dedicated_admins[@]}"
  do
    oc adm groups add-users dedicated-admins $username
  done

fi

echo "Waiting for new configuration to propagate to OpenShift OAuth pods."
sleep 10 # give the oauth-openshift deployment a chance to start rollout of new pods
until ! oc get deployment oauth-openshift -n openshift-authentication -o yaml | grep -q -e unavailableReplicas; do echo "\"oauth-openshift\" deployment is still updating, trying again in 10s"; sleep 10s; done
