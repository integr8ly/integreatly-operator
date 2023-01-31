#!/bin/sh
# Automatically backup and restore gcp postgres database instance, then verify the original database is consistent with the restored
# An existing postgres instance will be targeted, specified by the environment variable $INSTANCE_NAME
# The backups are saved to a cloud storage bucket relating to the postgres instance name
# example: INSTANCE_NAME=test-postgres ./test/scripts/backup-restore/gcp/postgres.sh

INFRA=$(oc get infrastructure cluster -o json)
CLUSTER_ID=$(jq -r '.status.infrastructureName' <<< $INFRA)
PROJECT_ID=$(jq -r '.status.platformStatus.gcp.projectID' <<< $INFRA)
REGION=$(jq -r '.status.platformStatus.gcp.region' <<< $INFRA)
BUCKET_NAME=$CLUSTER_ID-postgres-$INSTANCE_NAME
RESTORED_INSTANCE_NAME=$INSTANCE_NAME-restored
SERVICE_ACCOUNT=serviceAccount:$(gcloud sql instances describe $INSTANCE_NAME --project $PROJECT_ID --format json | jq -r '.serviceAccountEmailAddress')

# create cloud storage bucket for backups
gcloud storage buckets create gs://$BUCKET_NAME --project $PROJECT_ID --location $REGION
# allow the postgres service agent to use it for backups
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member $SERVICE_ACCOUNT --role roles/storage.objectAdmin
# trigger postgres database backup and save it in the bucket
gcloud sql export sql $INSTANCE_NAME gs://$BUCKET_NAME/original.sql --project $PROJECT_ID --database postgres
# prepare blank postgres database
gcloud sql instances create $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION --database-version POSTGRES_14 --tier db-f1-micro
# restore it to the original database state
SERVICE_ACCOUNT=serviceAccount:$(gcloud sql instances describe $RESTORED_INSTANCE_NAME --project $PROJECT_ID --format json | jq -r '.serviceAccountEmailAddress')
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member $SERVICE_ACCOUNT --role roles/storage.objectAdmin
gcloud sql import sql $RESTORED_INSTANCE_NAME gs://$BUCKET_NAME/original.sql --project $PROJECT_ID --database postgres --quiet
# create backup of the current state and save it in the bucket
gcloud sql export sql $INSTANCE_NAME gs://$BUCKET_NAME/current.sql --project $PROJECT_ID --database postgres
# copy the original and current database backups locally
gsutil cp gs://$BUCKET_NAME/original.sql original.sql
gsutil cp gs://$BUCKET_NAME/current.sql current.sql
# save the sorted database backup output to a text file
sort original.sql > original.txt
sort current.sql > current.txt
# compare the original and current state of backup files
if cmp --silent -- "original.txt" "current.txt"; then
  echo "postgres backups are identical"
else
  echo "postgres backups differ"
fi
# clean up resources
rm original.sql original.txt current.sql current.txt
gcloud storage rm --recursive gs://$BUCKET_NAME
gcloud sql instances delete $RESTORED_INSTANCE_NAME --project $PROJECT_ID --quiet