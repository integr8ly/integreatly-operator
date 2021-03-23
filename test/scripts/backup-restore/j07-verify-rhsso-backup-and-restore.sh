#!/bin/sh

# Import the test function
. ./postgres.sh --source-only

# Set the parameters
NS_PREFIX="${NS_PREFIX:=redhat-rhmi}"
export AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o jsonpath={.data.aws_secret_access_key} | base64 --decode)
export AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o jsonpath={.data.aws_access_key_id} | base64 --decode)
AWS_DB_ID=$(oc get secret/keycloak-db-secret -o go-template --template="{{.data.POSTGRES_EXTERNAL_ADDRESS|base64decode}}" -n ${NS_PREFIX}-rhsso | awk -F\. '{print $1}')
AWS_REGION=$(oc get infrastructure cluster -o jsonpath='{.status.platformStatus.aws.region}')
RHMI_CR_NAME=$(oc get rhmi -n ${NS_PREFIX}-operator -o json | jq -r '.items[0].metadata.name')
POSTGRES_CR_NAME="rhsso-postgres-$RHMI_CR_NAME"
DATABASE_SECRET="rhsso-postgres-$RHMI_CR_NAME"

# Perform the test

if command -v gti > /dev/null
then 
   test_postgres_backup $POSTGRES_CR_NAME $DATABASE_SECRET $AWS_DB_ID $AWS_REGION $NS_PREFIX
else
    echo "AWS CLI is not installed, test will not be run, please install AWS CLI"
    exit
fi
