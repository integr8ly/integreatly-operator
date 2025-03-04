#!/bin/sh
# USAGE
# ./j07-verify-rhsso-backup-and-restore <optional NS_PREFIX> <optional AWS_ACCESS_KEY_ID> <optional AWS_SECRET_ACCESS_KEY>
# ^C to break
# Tests RHSSO postgres back up and restore
#
# AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID should be set before running the script to support STS clusters as AWS
# AWS credentials are not available on STS clusters. By default the script will try to use the credentials on cluster.
#
# PREREQUISITES
# - oc (logged in at the cmd line)

# Import the test function
. ./postgres.sh --source-only

# Set the parameters
# Use passed in AWS credentials for STS, otherwise default to getting from cluster for normal OSD. Exit if these credentials are empty
export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:=$(oc get secret aws-creds -n kube-system -o jsonpath='{.data.aws_access_key_id}' | base64 --decode)}
[[ -z "$AWS_ACCESS_KEY_ID" ]] && { echo "AWS_ACCESS_KEY_ID cannot be empty"; exit 1; }

export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:=$(oc get secret aws-creds -n kube-system -o jsonpath='{.data.aws_secret_access_key}' | base64 --decode)}
[[ -z "$AWS_SECRET_ACCESS_KEY" ]] && { echo "AWS_SECRET_ACCESS_KEY cannot be empty"; exit 1; }

NS_PREFIX="${NS_PREFIX:=redhat-rhoam}"
AWS_DB_ID=$(oc get secret/keycloak-db-secret -o go-template --template="{{.data.POSTGRES_EXTERNAL_ADDRESS|base64decode}}" -n ${NS_PREFIX}-rhsso | awk -F\. '{print $1}')
AWS_REGION=$(oc get infrastructure cluster -o jsonpath='{.status.platformStatus.aws.region}')
RHMI_CR_NAME=$(oc get rhmi -n ${NS_PREFIX}-operator -o json | jq -r '.items[0].metadata.name')
POSTGRES_CR_NAME="rhsso-postgres-$RHMI_CR_NAME"
DATABASE_SECRET="rhsso-postgres-$RHMI_CR_NAME"

# Perform the test
test_postgres_backup $POSTGRES_CR_NAME $DATABASE_SECRET $AWS_DB_ID $AWS_REGION $NS_PREFIX
