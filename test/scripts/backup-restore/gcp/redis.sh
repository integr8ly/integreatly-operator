#!/bin/sh
# Automatically backup and restore gcp redis database instance, then verify the original database is consistent with the restored
# An existing redis instance will be targeted, specified by the environment variable $INSTANCE_NAME
# The backups are saved to a cloud storage bucket relating to the redis instance name
# example: INSTANCE_NAME=test-redis ./test/scripts/backup-restore/gcp/redis.sh

INFRA=$(oc get infrastructure cluster -o json)
CLUSTER_ID=$(jq -r '.status.infrastructureName' <<< $INFRA)
PROJECT_ID=$(jq -r '.status.platformStatus.gcp.projectID' <<< $INFRA)
REGION=$(jq -r '.status.platformStatus.gcp.region' <<< $INFRA)
BUCKET_NAME=$CLUSTER_ID-redis-$INSTANCE_NAME
RESTORED_INSTANCE_NAME=$INSTANCE_NAME-restored
SERVICE_ACCOUNT=$(gcloud redis instances describe $INSTANCE_NAME --region $REGION --format json | jq -r '.persistenceIamIdentity')

# create cloud storage bucket for backups
gcloud storage buckets create gs://$BUCKET_NAME --project $PROJECT_ID --location $REGION
# allow the redis service agent to use it for backups
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member $SERVICE_ACCOUNT --role roles/storage.admin
# trigger redis database backup and save it in the bucket
gcloud redis instances export gs://$BUCKET_NAME/original.rdb $INSTANCE_NAME --project $PROJECT_ID --region $REGION
# prepare blank redis database
gcloud redis instances create $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION
# restore it to the original database state
gcloud redis instances import gs://$BUCKET_NAME/original.rdb $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION
# create backup of the current state and save it in the bucket
gcloud redis instances export gs://$BUCKET_NAME/current.rdb $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION
# install rdbtools, python-lzf in order to parse redis database files
pip install rdbtools==0.1.15
pip install python-lzf==0.2.4
# copy the original and current database backups locally
gsutil cp gs://$BUCKET_NAME/original.rdb original.rdb
gsutil cp gs://$BUCKET_NAME/current.rdb current.rdb
# save the sorted database backup output to a text file
rdb --command diff original.rdb | sort > original.txt
rdb --command diff current.rdb | sort > current.txt
# compare the original and current state of backup files
if cmp --silent -- "original.txt" "current.txt"; then
  echo "redis backups are identical"
else
  echo "redis backups differ"
fi
# clean up resources
rm original.rdb original.txt current.rdb current.txt
gcloud storage rm --recursive gs://$BUCKET_NAME
gcloud redis instances delete $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION --quiet