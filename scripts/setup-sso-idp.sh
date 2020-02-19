#!/usr/bin/env bash
set -e
set -o pipefail 

PASSWORD="${PASSWORD:-$(openssl rand -base64 12)}"
REALM="${REALM:-testing-idp}"
INSTALLATION_PREFIX="${INSTALLATION_PREFIX:-$(oc get RHMIs --all-namespaces -o json | jq -r .items[0].spec.namespacePrefix)}"
INSTALLATION_PREFIX=${INSTALLATION_PREFIX%-} # remove trailing dash

# If CLUSTER_ID is not passed, find out ID based on currently targeted server
CLUSTER_ID="${CLUSTER_ID:-$(ocm get /api/clusters_mgmt/v1/clusters/ | jq -r ".items[] | select(.api.url == \"$(oc cluster-info | grep -Eo 'https?://[-a-zA-Z0-9\.:]*')\") | .id ")}"

echo "Cluster ID: $CLUSTER_ID"
echo "User password set to \"${PASSWORD}\""

KEYCLOAK_URL=https://$(oc get route keycloak-edge -n $INSTALLATION_PREFIX-rhsso -o json | jq -r .spec.host)
echo "Keycloak console: $KEYCLOAK_URL/auth/admin/master/console/#/realms/$REALM"
echo "Keycloack credentials: admin / $(oc get secret credential-rhsso -n $INSTALLATION_PREFIX-rhsso -o json | jq -r .data.ADMIN_PASSWORD | base64 --decode)"
echo "Keycloak realm: $REALM"

IDP_ID=$(ocm get "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers" | jq -r "select(.size > 0) | .items[] | select( .name == \"$REALM\") | .id")
if [[ ${IDP_ID} ]]; then
  echo "$REALM IDP is already present in OCM configuration."
  echo "OpenShift resources from testing-idp-template.yml will not be applied"
  echo "If you would like to re-apply any resources, delete the IDP from OCM and re-run this script."
  echo "To delete IDP execute: ocm delete \"/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers/$IDP_ID\""
else
  OAUTH_URL=https://$(oc get route oauth-openshift -n openshift-authentication -o json | jq -r .spec.host)
  CLIENT_SECRET=$(openssl rand -base64 20)

  oc process -p OAUTH_URL=$OAUTH_URL -p NAMESPACE=$INSTALLATION_PREFIX-rhsso -p REALM=$REALM -p PASSWORD=$PASSWORD -p CLIENT_SECRET=$CLIENT_SECRET -f ${BASH_SOURCE%/*}/testing-idp-template.yml | oc apply -f -

  sed "s|REALM|$REALM|g; s|KEYCLOAK_URL|$KEYCLOAK_URL|g; s|CLIENT_SECRET|$CLIENT_SECRET|g" "${BASH_SOURCE%/*}/ocm-idp-template.json" | ocm post /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers
  echo "$REALM IDP added into OCM configuration"
fi

echo "Adding users into dedicated-admins group:"
for username in "customer-admin01" "customer-admin02" "customer-admin03"
do
  if [[ $(ocm get "/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/groups/dedicated-admins/users" | jq ".items[] | select( .id == \"$username\")") ]]; then
    echo "$username is already in dedicated-admins group" 
  else
    echo '{"id":"'$username'"}' | ocm post /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/groups/dedicated-admins/users
    echo "$username added to dedicated-admins group"
  fi
done

until oc get oauth cluster -o json | jq .spec.identityProviders[].name | grep -q $REALM; do echo "\"cluster\" OAuth configuration does not contain our IDP yet, trying again in 10s"; sleep 10s; done
