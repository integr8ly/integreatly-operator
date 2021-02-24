#!/usr/bin/env bash
set -e
set -o pipefail

PASSWORD="${PASSWORD:-$(openssl rand -base64 12)}"
REALM="${REALM:-testing-idp}"
REALM_DISPLAY_NAME="${REALM_DISPLAY_NAME:-Testing IDP}"
INSTALLATION_PREFIX="${INSTALLATION_PREFIX:-$(oc get RHMIs --all-namespaces -o json | jq -r .items[0].spec.namespacePrefix)}"
INSTALLATION_PREFIX=${INSTALLATION_PREFIX%-} # remove trailing dash
ADMIN_USERNAME="${ADMIN_USERNAME:-customer-admin}"
DEDICATED_ADMIN_PASSWORD="${DEDICATED_ADMIN_PASSWORD:-$(openssl rand -base64 12)}"
NUM_ADMIN="${NUM_ADMIN:-3}"
REGULAR_USERNAME="${REGULAR_USERNAME:-test-user}"
NUM_REGULAR_USER="${NUM_REGULAR_USER:-10}"

# function to format user name depending on how many are created
format_user_name() {
  USER_NUM=$(printf "%02d" "$1") # Add leading zero to number
  USERNAME="$2$USER_NUM"         # Username combination of passed in username and number
}

# function to add admin to dedicated admin group via ocm or oc
add_user_to_dedicated_admin_group() {
  if [[ ${CLUSTER_ID} ]]; then # Add to dedicated admin group via ocm
    if [[ $(ocm get "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/groups/dedicated-admins/users" | jq ".items[] | select( .id == \"$1\")") ]]; then
      echo "$1 is already in dedicated-admins group"
    else
      echo '{"id":"'$1'"}' | ocm post "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/groups/dedicated-admins/users"
      echo "$1 added to dedicated-admins group"
    fi
  else # Add to dedicated admin group via oc
    oc adm groups add-users dedicated-admins "$1"
  fi
}

# Create dedicated admin users and add to dedicated admins group
create_dedicated_admins() {
  if ((NUM_ADMIN <= 0)); then
    echo "Skipping dedicated admin creation"
    return
  fi

  echo "Creating dedicated admin users"
  for ((i = 1; i <= NUM_ADMIN; i++)); do
    format_user_name $i "$ADMIN_USERNAME"
    oc process -p NAMESPACE="$INSTALLATION_PREFIX-rhsso" -p REALM="$REALM" -p PASSWORD="$DEDICATED_ADMIN_PASSWORD" -p USERNAME="$USERNAME" -p FIRSTNAME="Customer" -p LASTNAME="Admin ${USER_NUM}" -f "${BASH_SOURCE%/*}/admin-template.yml" | oc apply -f -
    add_user_to_dedicated_admin_group "$USERNAME"
  done
}

# Create sample normal users
create_regular_users() {
  if ((NUM_REGULAR_USER <= 0)); then
    echo "Skipping regular user creation"
    return
  fi

  echo "Creating regular users"
  for ((i = 1; i <= NUM_REGULAR_USER; i++)); do
    format_user_name $i "$REGULAR_USERNAME"
    oc process -p NAMESPACE="$INSTALLATION_PREFIX-rhsso" -p REALM="$REALM" -p PASSWORD="$PASSWORD" -p USERNAME="$USERNAME" -p FIRSTNAME="Test" -p LASTNAME="User ${USER_NUM}" -f "${BASH_SOURCE%/*}/user-template.yml" | oc apply -f -
  done
}

create_users() {
  create_dedicated_admins
  create_regular_users
}

echo "User password set to \"${PASSWORD}\""
echo "Dedciated Admin password set to \"${DEDICATED_ADMIN_PASSWORD}\""
CLIENT_SECRET=$(openssl rand -base64 20)
OAUTH_URL=https://$(oc get route oauth-openshift -n openshift-authentication -o json | jq -r .spec.host)
KEYCLOAK_URL=https://$(oc get route keycloak-edge -n "$INSTALLATION_PREFIX-rhsso" -o json | jq -r .spec.host)
echo "Keycloak console: $KEYCLOAK_URL/auth/admin/master/console/#/realms/$REALM"
echo "Keycloack credentials: admin / $(oc get secret credential-rhsso -n "$INSTALLATION_PREFIX-rhsso" -o json | jq -r .data.ADMIN_PASSWORD | base64 --decode)"
echo "Keycloak realm: $REALM"

# If CLUSTER_ID is not passed, find out ID based on currently targeted server
set +e # ignore errors in environments without ocm command
CLUSTER_ID="${CLUSTER_ID:-$(ocm get clusters --parameter search="api.url like '$(oc cluster-info | grep -Eo 'https?://[-a-zA-Z0-9\.:]*')'" 2>/dev/null | jq -r .items[0].id)}"
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

    # Delete any keycloak client of the same name to allow regenerating correct client secret for keycloak client
    oc delete keycloakclient "$REALM-client" -n "$INSTALLATION_PREFIX-rhsso" --ignore-not-found=true

    # apply KeycloakRealm and KeycloakClient from a template
    oc process -p OAUTH_URL="$OAUTH_URL" -p NAMESPACE="$INSTALLATION_PREFIX-rhsso" -p REALM="$REALM" -p REALM_DISPLAY_NAME="$REALM_DISPLAY_NAME" -p CLIENT_SECRET="$CLIENT_SECRET" -f "${BASH_SOURCE%/*}/testing-idp-template.yml" | oc apply -f -

    sed "s|REALM|$REALM|g; s|KEYCLOAK_URL|$KEYCLOAK_URL|g; s|CLIENT_SECRET|$CLIENT_SECRET|g" "${BASH_SOURCE%/*}/ocm-idp-template.json" | ocm post "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers"
    echo "$REALM IDP added into OCM configuration"
  fi

  # create KeycloakUsers
  create_users

  until oc get oauth cluster -o json | jq .spec.identityProviders[].name | grep -q "$REALM"; do
    echo "\"cluster\" OAuth configuration does not contain our IDP yet, trying again in 10s"
    sleep 10s
  done

else
  #  If CLUSETER_ID is not detected - manage IDP directly in cluster config
  echo "No CLUSTER_ID detected. IDP will be added directly into \"cluster\" OAuth resource."

  if [[ ! $(oc get group dedicated-admins) ]]; then
    # create dedicated-admins group if it doesn't exist
    echo '{"kind": "Group", "apiVersion": "user.openshift.io/v1", "metadata": { "name": "dedicated-admins" }, "users": null }' | oc create -f -
  fi

  # Delete any keycloak client of the same name to allow regenerating correct client secret for keycloak client
  oc delete keycloakclient "$REALM-client" -n "$INSTALLATION_PREFIX-rhsso" --ignore-not-found=true

  # apply KeycloakRealm and KeycloakClient from a template
  oc process -p OAUTH_URL="$OAUTH_URL" -p NAMESPACE="$INSTALLATION_PREFIX-rhsso" -p REALM="$REALM" -p REALM_DISPLAY_NAME="$REALM_DISPLAY_NAME" -p CLIENT_SECRET="$CLIENT_SECRET" -f "${BASH_SOURCE%/*}/testing-idp-template.yml" | oc apply -f -
  # create KeycloakUsers
  create_users

  # update the Secret used by OpenShift OAuth server
  CLIENT_SECRET_NAME="idp-$REALM"
  oc delete secret "$CLIENT_SECRET_NAME" -n openshift-config 2>/dev/null || true
  oc create secret generic "$CLIENT_SECRET_NAME" --from-literal=clientSecret="$CLIENT_SECRET" -n openshift-config

  CA_CONFIGMAP_NAME=""
  # shellcheck disable=SC2143
  if [[ $(curl -sSo /dev/null "$KEYCLOAK_URL" 2>&1 | grep "self signed certificate") ]]; then
    # update the Secret that stores CA cert
    CA_CONFIGMAP_NAME="idp-ca-$REALM"
    oc get secret router-ca -o yaml -n openshift-ingress-operator -o 'go-template={{index .data "tls.crt"}}' | base64 --decode >router-ca.tmp
    oc delete configmap "$CA_CONFIGMAP_NAME" -n openshift-config 2>/dev/null || true
    oc create configmap "$CA_CONFIGMAP_NAME" --from-file=ca.crt=router-ca.tmp -n openshift-config
  fi

  # shellcheck disable=SC2143
  if [[ ! $(oc get oauth cluster -o yaml | grep identityProviders) ]]; then
    # add an empty array into .spec.identityProviders of "cluster" OAuth resource if it was set to null
    oc patch oauth cluster --type=json -p '[{"op":"add", "path":"/spec/identityProviders", "value":[]}]'
  fi
  # try to add the testing IDP .spec.identityProviders of "cluster" OAuth resource
  if [[ $(oc patch oauth cluster --type=json -p '[{"op":"add", "path":"/spec/identityProviders/-", "value":{ "name": "'$REALM'", "mappingMethod": "claim", "type": "OpenID", "openID": { "ca": {"name": "'$CA_CONFIGMAP_NAME'"}, "clientID": "openshift", "clientSecret": { "name": "'$CLIENT_SECRET_NAME'" }, "issuer": "'$KEYCLOAK_URL'/auth/realms/'$REALM'", "claims": { "preferredUsername": ["preferred_username"], "email": ["email"], "name": ["name"] } } } }]' 2>&1) =~ "must have a unique name" ]]; then
    echo "$REALM IDP already exists in \"cluster\" OAuth, this resource will not be updated!"
  fi

fi

echo "Waiting for new configuration to propagate to OpenShift OAuth pods."
sleep 10 # give the oauth-openshift deployment a chance to start rollout of new pods
until ! oc get deployment oauth-openshift -n openshift-authentication -o yaml | grep -q -e unavailableReplicas; do
  echo "\"oauth-openshift\" deployment is still updating, trying again in 10s"
  sleep 10s
done
