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

draw_horizontal_line () { printf '%.s─' $(seq 1 $(tput cols)); }

echo "creating cloud storage bucket for postgres backups..."
gcloud storage buckets create gs://$BUCKET_NAME --project $PROJECT_ID --location $REGION

draw_horizontal_line
echo "allowing postgres instance '$INSTANCE_NAME' service agent to manage bucket '$BUCKET_NAME'..."
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member $SERVICE_ACCOUNT --role roles/storage.admin

draw_horizontal_line
echo "creating backup of postgres instance '$INSTANCE_NAME' and saving it to bucket '$BUCKET_NAME'..."
gcloud sql export sql $INSTANCE_NAME gs://$BUCKET_NAME/original.sql --project $PROJECT_ID --database postgres

draw_horizontal_line
echo "preparing blank postgres instance '$RESTORED_INSTANCE_NAME'..."
gcloud sql instances create $RESTORED_INSTANCE_NAME --project $PROJECT_ID --region $REGION --database-version POSTGRES_14 --tier db-f1-micro

draw_horizontal_line
echo "allowing postgres instance '$RESTORED_INSTANCE_NAME' service agent to manage bucket '$BUCKET_NAME'..."
SERVICE_ACCOUNT=serviceAccount:$(gcloud sql instances describe $RESTORED_INSTANCE_NAME --project $PROJECT_ID --format json | jq -r '.serviceAccountEmailAddress')
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member $SERVICE_ACCOUNT --role roles/storage.admin

draw_horizontal_line
echo "restoring backup to postgres instance '$RESTORED_INSTANCE_NAME'..."
gcloud sql import sql $RESTORED_INSTANCE_NAME gs://$BUCKET_NAME/original.sql --project $PROJECT_ID --database postgres --quiet

draw_horizontal_line
echo "creating backup of postgres instance '$RESTORED_INSTANCE_NAME' and saving it to bucket '$BUCKET_NAME'..."
gcloud sql export sql $RESTORED_INSTANCE_NAME gs://$BUCKET_NAME/current.sql --project $PROJECT_ID --database postgres

draw_horizontal_line
echo "downloading postgres backups from cloud storage bucket '$BUCKET_NAME'..."
gsutil cp gs://$BUCKET_NAME/original.sql original.sql
gsutil cp gs://$BUCKET_NAME/current.sql current.sql

draw_horizontal_line
echo "comparing the original and current postgres backup files..."
sort original.sql > original.txt
sort current.sql > current.txt
if cmp --silent -- "original.txt" "current.txt"; then
  echo "postgres backups are identical!"
else
  echo "postgres backups differ!"
fi

draw_horizontal_line
echo "cleaning up temporary files, cloud storage bucket '$BUCKET_NAME' and postgres instance '$RESTORED_INSTANCE_NAME'..."
rm original.sql original.txt current.sql current.txt
gcloud storage rm --recursive gs://$BUCKET_NAME
gcloud sql instances delete $RESTORED_INSTANCE_NAME --project $PROJECT_ID --quiet

draw_horizontal_line
echo 'test completed successfully!'
