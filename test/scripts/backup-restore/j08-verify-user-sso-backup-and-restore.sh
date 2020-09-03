#!/bin/sh

# Import the test function
. ./postgres.sh --source-only

# Set the parameters
AWS_DB_ID=$(oc get secret/keycloak-db-secret -o go-template --template="{{.data.POSTGRES_EXTERNAL_ADDRESS|base64decode}}" -n redhat-rhmi-user-sso | awk -F\. '{print $1}')
POSTGRES_CR_NAME=rhssouser-postgres-rhmi
DATABASE_SECRET=rhssouser-postgres-rhmi

# Perform the test
test_postgres_backup $POSTGRES_CR_NAME $DATABASE_SECRET $AWS_DB_ID