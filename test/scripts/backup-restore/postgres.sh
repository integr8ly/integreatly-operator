#!/bin/sh

# Automatically perform a backup and restore of a product Postgres database and
# verify that the data in the database is consistent after the restoration.
#
# It receives the following parameters:
#   1. Name of the Postgres CR
#   2. Name of the secret with the credentials to the database. This secret must be
#      in the redhat-rhmi-operator namespace, and contain the following data:
#      * host
#      * port
#      * database
#      * username
#      * password
#   3. ID of the AWS RDS instance.
#
# In order to use this function, source this file and pass call it with the
# parameters. Example:
#
# ```
# . ./postgres.sh --source-only
# test_postgres_backup $POSTGRES_CR_NAME $DATABASE_SECRET $AWS_DB_ID
# ````
#
# It will print the test log in the standard output, so it's recommended to
# save it for future evaluation.
test_postgres_backup () {
  # Get the function parameters
  POSTGRES_CR_NAME=$1
  POSTGRES_SECRET=$2
  AWS_DB_ID=$3
  AWS_REGION=$4

  # Get the database credentials
  DB_HOST=`oc get secrets/$POSTGRES_SECRET -n redhat-rhmi-operator -o template --template={{.data.host}} | base64 -d`
  DB_PORT=`oc get secrets/$POSTGRES_SECRET -n redhat-rhmi-operator -o template --template={{.data.port}} | base64 -d`
  DATABASE_NAME=`oc get secrets/$POSTGRES_SECRET -n redhat-rhmi-operator -o template --template={{.data.database}} | base64 -d`
  DB_USER=`oc get secrets/$POSTGRES_SECRET -n redhat-rhmi-operator -o template --template={{.data.username}} | base64 -d`
  DB_PASSWORD=`oc get secrets/$POSTGRES_SECRET -n redhat-rhmi-operator -o template --template={{.data.password}} | base64 -d`

  # Disable the AWS pager to avoid the user to be displayed with an interactive
  # output when using the AWS CLI
  AWS_PAGER=""

  # Get a dump of the database before the deletion
  echo "Dumping current database..."
  dump_database dump_before.sql
  echo "Dumped original database to dump_before.sql"

  # Create the snapshot
  POSTGRES_SNAPSHOT_NAME="${POSTGRES_CR_NAME}-snapshot-test-$(date +"%Y-%m-%d-%H%M%S")"
  echo "Creating snapshot $POSTGRES_SNAPSHOT_NAME..."
  cat << EOF | oc create -f - -n redhat-rhmi-operator
  apiVersion: integreatly.org/v1alpha1
  kind: PostgresSnapshot
  metadata:
    # Needs to be a unique name for this snapshot
    name: $POSTGRES_SNAPSHOT_NAME
  spec:
    # The Postgres resource name for the snapshot you want to take
    resourceName: $POSTGRES_CR_NAME
EOF

  # Wait for it to complete
  while true
  do
      PHASE=`oc get postgressnapshot/$POSTGRES_SNAPSHOT_NAME -n redhat-rhmi-operator -o template --template={{.status.phase}}`
      if [ "$PHASE" = 'complete' ]; then
        echo "Snapshot creation completed."
        break
      fi

      echo "Waiting for snapshot to complete. Current phase: $PHASE..."
      sleep 10s
  done

  # Edit Postgres CR to prevent RDS recreation during restoration
  echo "Disabling automatic RDS recreation..."
  oc patch postgres/$POSTGRES_CR_NAME -n redhat-rhmi-operator -p '{"spec":{"skipCreate":true}}' --type merge

  # Get VPC security group IDs from existing RDS
  VPC_SECURITY_GROUP_IDS=$(aws rds describe-db-instances --db-instance-identifier $AWS_DB_ID --region $AWS_REGION | jq '.DBInstances[0].VpcSecurityGroups[].VpcSecurityGroupId' -r | tr '\n' ' ' | sed -e 's/[[:space:]]$//')
  echo "Obtained VPC Security Group IDs: $VPC_SECURITY_GROUP_IDS"

  # Get Subnet group name from existing RDS
  DB_SUBNET_GROUP_NAME=$(aws rds describe-db-instances --db-instance-identifier $AWS_DB_ID --region $AWS_REGION | jq '.DBInstances[0].DBSubnetGroup.DBSubnetGroupName' -r)
  echo "Obtained Subnet group name: $DB_SUBNET_GROUP_NAME"

  echo "Removing Postgres instance deletion protection..."
  aws rds modify-db-instance \
      --db-instance-identifier $AWS_DB_ID --no-deletion-protection --region $AWS_REGION > /dev/null

  echo "Deleting Postgres instance..."
  aws rds delete-db-instance \
      --db-instance-identifier $AWS_DB_ID \
      --skip-final-snapshot \
      --no-delete-automated-backups \
      --region $AWS_REGION > /dev/null

  while true
  do
    # Check if the database still exists. If it does not, break the loop
    EXISTS=`aws rds describe-db-instances --region $AWS_REGION | jq ".DBInstances | any(.DBInstanceIdentifier == \"$AWS_DB_ID\")"`
    if [ "$EXISTS" = 'false' ]; then
      echo "Database deleted"
      break
    fi

    # Attempt to get the database status. If it fails, check if the error is
    # not found. If it's not found it means the database was deleted, so break
    # the loop. Otherwise report the error
    DATABASE=`aws rds describe-db-instances --db-instance-identifier $AWS_DB_ID --region $AWS_REGION 2>&1`
    if [ ! "$?" = 0 ]; then
      if echo $DATABASE | grep -q "DBInstanceNotFound"; then
        echo "Database deleted"
        break
      fi

      echo "Unexpected error requesting database: $DATABASE"
      exit 1
    fi

    # Assert that, as the database hasn't been deleted yet, the status is "deleting"
    # meanwhile
    STATUS=`echo $DATABASE | jq -r '.DBInstances[0].DBInstanceStatus'`
    if [ "$STATUS" = 'deleting' ]; then
      echo "Waiting for database deletion..."
      sleep 10s
      continue
    fi

    # If the database still exists but the status is not deleting, fail the test
    echo "Unexpected status '$STATUS' when deleting database"
    exit 1
  done

  # Restore the database
  echo "Restoring database from snapshot..."
  RDS_RESTORE_SNAPSHOT=`oc get postgressnapshots $POSTGRES_SNAPSHOT_NAME -n redhat-rhmi-operator -o json | jq -r '.status.snapshotID'`
  aws rds restore-db-instance-from-db-snapshot \
      --db-instance-identifier $AWS_DB_ID \
      --db-snapshot-identifier $RDS_RESTORE_SNAPSHOT \
      --db-subnet-group-name $DB_SUBNET_GROUP_NAME \
      --vpc-security-group-ids $VPC_SECURITY_GROUP_IDS \
      --multi-az \
      --deletion-protection \
      --region $AWS_REGION > /dev/null


  # Wait for the database to be available
  while true
  do
    STATUS=`aws rds describe-db-instances --db-instance-identifier $AWS_DB_ID --region $AWS_REGION | jq -r '.DBInstances[0].DBInstanceStatus'`
    if [ "$STATUS" = 'available' ]; then
      echo "Database restored."
      break
    fi

    echo "Waiting for snapshot restoration... Current status: $STATUS"
    sleep 10s
  done

  # Restore default scaling options that couldn't be added as part of the restore
  echo "Restoring default scaling options..."
  aws rds modify-db-instance \
      --db-instance-identifier $AWS_DB_ID --max-allocated-storage 100 \
      --region $AWS_REGION > /dev/null

  # Revert PostGres CR Change
  echo "Re-enabling automating RDS recreation..."
  oc patch postgres/$POSTGRES_CR_NAME -n redhat-rhmi-operator -p '{"spec":{"skipCreate":false}}' --type merge

  # Dump the database after the restoration
  echo "Dumping restored database..."
  dump_database dump_after.sql
  echo "Dumped restore database to dump_after.sql"

  echo "Calculating difference between databases..."
  DB_DIFF=`diff dump_before.sql dump_after.sql`
  if [ ! -z "$DB_DIFF" ]; then
    echo "Difference found between database dumps:"
    echo $DB_DIFF

    exit 1
  fi

  echo "No difference between restored database."
  echo "Deleting database dumps."
  rm dump_before.sql
  rm dump_after.sql

  echo "Test finished successfully."
}

dump_database() {
  DUMP_FILE=$1

  echo "Creating throwaway Postgres container..."
  cat << EOF | oc create -f - -n redhat-rhmi-operator
  apiVersion: integreatly.org/v1alpha1
  kind: Postgres
  metadata:
    name: throw-away-postgres
    labels:
      productName: productName
  spec:
    secretRef:
      name: throw-away-postgres-sec
    tier: development
    type: workshop
EOF
  
  # Wait for the Postgres to be reconciled
  while true
  do
      PHASE=`oc get postgres/throw-away-postgres -n redhat-rhmi-operator -o template --template={{.status.phase}}`
      if [ "$PHASE" = 'complete' ]; then
        break
      fi

      echo "Waiting for throwaway Postgres container to complete. Current phase: $PHASE..."
      sleep 10s
  done

  # Create the dump
  kubectl exec deploy/throw-away-postgres \
    -n redhat-rhmi-operator \
    -- env PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER $DATABASE_NAME > $DUMP_FILE

  # Delete the throwaway postgres
  oc delete postgres/throw-away-postgres -n redhat-rhmi-operator
}
