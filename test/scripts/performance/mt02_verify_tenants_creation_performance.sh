#!/bin/bash

# The script could be used for Multitenants Performance verification
# script could be used together with multitenancy_performance.go, to provide flexibility for testing
# Notes:
#   IDP should be created before script execution.
#       NUM_REGULAR_USER in scripts/setup-sso-idp.sh need to be changed before running IDP creation
#   Promotions options in the script are not checked yet and could require updating. Use PROMOTE_STAGE=false and PROMOTE_PROD=false
#   Recommended to use multitenancy_performance.go for Cluster users creation, as usersLogin function is not creating Cluster users_
# Script require improvement, but could be used for performance investigation

ACCESS_TOKEN="xxxxxxx"
ADMIN_URL="https://3scale-admin.apps.xxxx.xx.s1.devshift.org"
CLUSTER_WIDLCARD_URL="xxxxx.xxx.s1.devshift.org"
INSTALL_TYPE_PREFIX="sandbox"
USER_BASENAME="test-user"
USER_PASSWD="Password1"
ADMIN_API_TOKEN="sha256~#####"
MIN=1
MAX=200
CREATE_CRS=true
CREATE_PROD=true
PROMOTE_STAGE=false
PROMOTE_PROD=false
DO_USERS_LOGIN=false

createTenantCR() {
  cat << EOF | oc create -f -
apiVersion: integreatly.org/v1alpha1
kind: APIManagementTenant
metadata:
  name: example
  namespace: "$USER_BASENAME$i-dev"
spec: {}
EOF
}

createResourceUsageReport(){
  oc adm top pods -n "$INSTALL_TYPE_PREFIX-rhoam-3scale" > ./ResourceUsageReport
  oc adm top pods -n "$INSTALL_TYPE_PREFIX-rhoam-3scale-operator" >> ./ResourceUsageReport
}

usersLogin () {
  echo "IDP Users login to create sso users"
  for i in $(seq $MIN $MAX)
  do
    # login with user i
    if [[ $i -lt 10 ]]
    then
      i="0$i"
    fi
    oc login -u "$USER_BASENAME$i" -p $USER_PASSWD >/dev/null
    echo "user "$USER_BASENAME$i" logged in"
    sleep 5
  done
}

adminCreateTenantCRs () {
  echo "Create projects and Tenant CRs"
  #oc login --token=$ADMIN_API_TOKEN --server=$SERVER_URL >/dev/null
  for i in $(seq $MIN $MAX)
  do
    if [[ $i -lt 10 ]]
    then
      i="0$i"
    fi
    echo "Create $USER_BASENAME$i-dev project and APIManagementTenant instance"
    oc new-project "$USER_BASENAME$i-dev" >/dev/null
    createTenantCR&
    sleep 1
  done
}

# Create products using 3scale-admin route and admin access token. It's simplified approach
# To improve the test - need create tenant Products using access tokens and routes of specific tenant users
createProducts () {
  echo "### Create Products ###"
  for i in $(seq $MIN $MAX)
  do
    #USER_URL="https://$USER_BASENAME$i.$CLUSTER_WIDLCARD_URL"
    curl -v  -X POST "$ADMIN_URL/admin/api/services.xml" -d "access_token=$ACCESS_TOKEN&name=Product$i"
  done
}

## to check
promoteProductsToStaging () {
  for i in $(seq $MIN $MAX)
  do
    prod_id=$(expr $i + 2)
    backend_api_id=$(expr $i + 5)
    curl -v -X POST "$ADMIN_URL/admin/api/services/$prod_id/backend_usages.json" -d "access_token=$ACCESS_TOKEN&backend_api_id=$backend_api_id&path=%2F"
    curl -v -X POST "$ADMIN_URL/admin/api/services/$prod_id/proxy/deploy.xml" -d "access_token=$ACCESS_TOKEN"
  done
}

## to check
promoteProductsToProduction () {
  for i in $(seq $MIN $MAX)
  do
    prod_id=$(expr $i + 2)
    curl -v -X POST "{$ADMIN_URL}/admin/api/services/$prod_id/proxy/configs/sandbox/1/promote.json" -d "access_token={$ACCESS_TOKEN}&to=production"
  done
}

## main:
echo "Create $MAX tenants"
if $DO_USERS_LOGIN
then
  usersLogin
fi

if $CREATE_CRS
then
  adminCreateTenantCRs
fi

#createResourceUsageReport

if $CREATE_PROD
then
  createProducts
fi

if $PROMOTE_STAGE
then
  promoteProductsToStaging
fi

if $PROMOTE_PROD
then
  promoteProductsToProduction
fi
