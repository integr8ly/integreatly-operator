#!/bin/sh

# Import the test function
. ./postgres.sh --source-only

# Make sure we're using GNU grep
if grep -V | grep BSD; then
  grep_cmd=$(which grep)
  if [ -z "$grep_cmd" ]; then
    echo "This script requires GNU version of 'grep'. You can install it with \`brew install grep\`"
    exit 1
  fi
else
  grep_cmd=$(which grep)
fi

# Set the parameters
NS_PREFIX="${NS_PREFIX:=redhat-rhoam}"
export AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o jsonpath='{.data.aws_secret_access_key}' | base64 --decode)
export AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o jsonpath='{.data.aws_access_key_id}' | base64 --decode)
AWS_DB_ID=$(oc get secret/system-database -o go-template --template="{{.data.URL|base64decode}}" -n ${NS_PREFIX}-3scale | $grep_cmd -Po "(?<=@).*?(?=\.)")
AWS_REGION=$(oc get infrastructure cluster -o jsonpath='{.status.platformStatus.aws.region}')
RHMI_CR_NAME=$(oc get rhmi -n ${NS_PREFIX}-operator -o json | jq -r '.items[0].metadata.name')
POSTGRES_CR_NAME="threescale-postgres-$RHMI_CR_NAME"
DATABASE_SECRET="threescale-postgres-$RHMI_CR_NAME"

# Perform the test
test_postgres_backup $POSTGRES_CR_NAME $DATABASE_SECRET $AWS_DB_ID $AWS_REGION $NS_PREFIX