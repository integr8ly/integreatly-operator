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

draw_horizontal_line () { printf '%.s─' $(seq 1 $(tput cols)); }

echo "creating cloud storage bucket for redis backups..."
gcloud storage buckets create gs://$BUCKET_NAME --project $PROJECT_ID --location $REGION

draw_horizontal_line
echo "allowing the redis service agent to manage bucket '$BUCKET_NAME'..."
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member $SERVICE_ACCOUNT --role roles/storage.admin

draw_horizontal_line
echo "creating backup of redis instance '$INSTANCE_NAME' and saving it to bucket '$BUCKET_NAME'..."
gcloud redis instances export gs://$BUCKET_NAME/original.rdb $INSTANCE_NAME --project $PROJECT_ID --region $REGION

draw_horizontal_line
echo "preparing blank redis instance '$RESTORED_INSTANCE_NAME'..."
gcloud redis instances create $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION

draw_horizontal_line
echo "restoring backup to redis instance '$RESTORED_INSTANCE_NAME'..."
gcloud redis instances import gs://$BUCKET_NAME/original.rdb $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION

draw_horizontal_line
echo "creating backup of redis instance '$RESTORED_INSTANCE_NAME' and saving it to bucket '$BUCKET_NAME'..."
gcloud redis instances export gs://$BUCKET_NAME/current.rdb $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION

draw_horizontal_line
echo "installing rdbtools, python-lzf in order to parse redis database files..."
pip install rdbtools==0.1.15
pip install python-lzf==0.2.4

draw_horizontal_line
echo "downloading redis backups from cloud storage bucket '$BUCKET_NAME'..."
gsutil cp gs://$BUCKET_NAME/original.rdb original.rdb
gsutil cp gs://$BUCKET_NAME/current.rdb current.rdb

draw_horizontal_line
echo "comparing the original and current redis backup files..."
rdb --command diff original.rdb | sort > original.txt
rdb --command diff current.rdb | sort > current.txt
if cmp --silent -- "original.txt" "current.txt"; then
  echo "redis backups are identical!"
else
  echo "redis backups differ!"
fi

draw_horizontal_line
echo "cleaning up temporary files, cloud storage bucket '$BUCKET_NAME' and redis instance '$RESTORED_INSTANCE_NAME'..."
rm original.rdb original.txt current.rdb current.txt
gcloud storage rm --recursive gs://$BUCKET_NAME
gcloud redis instances delete $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION --quiet

draw_horizontal_line
echo 'test completed successfully!'
