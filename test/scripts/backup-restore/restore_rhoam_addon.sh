#!/bin/bash

# ==============================================================================
# RHOAM ADDON - AUTOMATED DISASTER RECOVERY RESTORE SCRIPT (Interactive/Batch)
# ==============================================================================
# This script automates the full restore process for a RHOAM AddOn installation.
# It reads all necessary parameters from the 'restore_rhoam_addon_config.sh' file.
#
# USAGE:
#   - Run the script: ./restore_rhoam_addon.sh
#   - At the prompt, choose [B]atch for a fully automatic run or
#     [I]nteractive to confirm each individual restore operation.
# ==============================================================================

set -e
set -o pipefail

# --- LOAD CONFIGURATION ---
# Ensure restore_rhoam_addon_config.sh is in the same directory or provide the full path.
if [ ! -f ./restore_rhoam_addon_config.sh ]; then
    echo "ERROR: Configuration file restore_rhoam_addon_config.sh not found." >&2
    exit 1
fi
source ./restore_rhoam_addon_config.sh
# --- END OF CONFIGURATION ---


# --- HELPER FUNCTIONS ---
log() {
  echo
  echo "------------------------------------------------------------------"
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] - $1"
  echo "------------------------------------------------------------------"
}

restore_rds_instance() {
  local db_id=$1
  local snapshot_id=$2
  log "Processing RDS Instance: $db_id"

  log "Capturing config from snapshot $snapshot_id..."
  local snapshot_info=$(aws rds describe-db-snapshots --db-snapshot-identifier "$snapshot_id" --region "$AWS_REGION" --query "DBSnapshots[0]")
  local instance_class=$(echo "$snapshot_info" | jq -r '.DBInstanceClass')
  local subnet_group=$(echo "$snapshot_info" | jq -r '.DBSubnetGroupName')
  local vpc_sgs=$(echo "$snapshot_info" | jq -r '.VpcSecurityGroupIds | join(" ")')
  local is_multi_az=$(echo "$snapshot_info" | jq -r '.MultiAZ')
  local multi_az_flag=""

  if [[ -z "$instance_class" || "$instance_class" == "null" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the DB Instance Class from the snapshot."
    read -p "Enter DB Instance Class (e.g., db.m5.large): " instance_class
    if [[ -z "$instance_class" ]]; then echo "Aborting."; exit 1; fi
  fi

  if [[ -z "$subnet_group" || "$subnet_group" == "null" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the DB Subnet Group from the snapshot."
    read -p "Enter DB Subnet Group Name: " subnet_group
    if [[ -z "$subnet_group" ]]; then echo "Aborting."; exit 1; fi
  fi

  if [[ -z "$vpc_sgs" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the VPC Security Groups from the snapshot."
    read -p "Enter VPC Security Group IDs: " vpc_sgs
    if [[ -z "$vpc_sgs" ]]; then echo "Aborting."; exit 1; fi
  fi

  if [[ "$is_multi_az" == "true" ]]; then
    multi_az_flag="--multi-az"
  elif [[ "$is_multi_az" == "false" ]]; then
    multi_az_flag="--no-multi-az"
  else
    echo; log "⚠️ WARNING: Could not automatically determine the Multi-AZ status from the snapshot."
    read -p "Do you want the restored database to be Multi-AZ? (yes/no): " choice
    case "$choice" in
      y|Y|yes|Yes ) multi_az_flag="--multi-az";;
      n|N|no|No ) multi_az_flag="--no-multi-az";;
      * ) echo "Invalid choice. Aborting."; exit 1;;
    esac
  fi

  log "Checking existence of RDS instance $db_id..."
  if aws rds describe-db-instances --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager > /dev/null 2>&1; then
    log "Instance found. Proceeding with deletion..."
    aws rds modify-db-instance --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-deletion-protection --apply-immediately --no-cli-pager
    aws rds delete-db-instance --db-instance-identifier "$db_id" --region "$AWS_REGION" --skip-final-snapshot --no-cli-pager
    aws rds wait db-instance-deleted --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager
  else
    echo; log "⚠️ WARNING: RDS instance '$db_id' not found."
    read -p "Do you want to [S]kip deletion and proceed with restore, or [A]bort? (S/A): " choice
    case "$choice" in
      s|S ) echo "Skipping deletion...";;
      a|A ) echo "Aborting."; exit 1;;
      * ) echo "Invalid choice. Aborting."; exit 1;;
    esac
  fi

  log "Restoring $db_id from snapshot $snapshot_id..."
  aws rds restore-db-instance-from-db-snapshot \
    --db-instance-identifier "$db_id" --db-snapshot-identifier "$snapshot_id" --region "$AWS_REGION" \
    --db-instance-class "$instance_class" --db-subnet-group-name "$subnet_group" --vpc-security-group-ids "$vpc_sgs" \
    $multi_az_flag --deletion-protection --no-cli-pager
  aws rds wait db-instance-available --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager
  log "✅ RDS instance $db_id restored."
}

restore_redis_instance() {
  local redis_id=$1
  local snapshot_id=$2
  log "Processing Redis Replication Group: $redis_id"

  log "Capturing config from snapshot $snapshot_id..."
  local snapshot_info=$(aws elasticache describe-snapshots --snapshot-name "$snapshot_id" --region "$AWS_REGION" --query "Snapshots[0]")
  local engine=$(echo "$snapshot_info" | jq -r '.Engine')
  local engine_version=$(echo "$snapshot_info" | jq -r '.EngineVersion')
  local subnet_group=$(echo "$snapshot_info" | jq -r '.NodeSnapshots[0].CacheSubnetGroupName')

  if [[ -z "$engine" || "$engine" == "null" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the Engine from the snapshot."
    read -p "Enter the engine (e.g., redis): " engine
    if [[ -z "$engine" ]]; then echo "Aborting."; exit 1; fi
  fi

  if [[ -z "$engine_version" || "$engine_version" == "null" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the Engine Version from the snapshot."
    read -p "Enter the engine version (e.g., 7.1): " engine_version
    if [[ -z "$engine_version" ]]; then echo "Aborting."; exit 1; fi
  else
    # If version is found, truncate it to major.minor (e.g., 7.1.0 -> 7.1)
    engine_version=$(echo "$engine_version" | cut -d'.' -f1,2)
  fi

  if [[ -z "$subnet_group" || "$subnet_group" == "null" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the Cache Subnet Group from the snapshot."
    echo "(To list available groups, run this in another terminal: aws elasticache describe-cache-subnet-groups --region $AWS_REGION)"
    read -p "Enter Cache Subnet Group Name: " subnet_group
    if [[ -z "$subnet_group" ]]; then echo "Aborting."; exit 1; fi
  fi

  log "Checking existence of Redis replication group $redis_id..."
  if aws elasticache describe-replication-groups --replication-group-id "$redis_id" --region "$AWS_REGION" --no-cli-pager > /dev/null 2>&1; then
      log "Replication group found. Proceeding with deletion..."
      aws elasticache delete-replication-group --replication-group-id "$redis_id" --region "$AWS_REGION" --no-cli-pager
      aws elasticache wait replication-group-deleted --replication-group-id "$redis_id" --region "$AWS_REGION" --no-cli-pager
  else
      echo; log "⚠️ WARNING: Redis replication group '$redis_id' not found."
      read -p "Do you want to [S]kip deletion and proceed with restore, or [A]bort? (S/A): " choice
      case "$choice" in
        s|S ) echo "Skipping deletion...";;
        a|A ) echo "Aborting."; exit 1;;
        * ) echo "Invalid choice. Aborting."; exit 1;;
      esac
  fi

  log "Restoring $redis_id from snapshot $snapshot_id..."
  aws elasticache create-replication-group \
    --replication-group-id "$redis_id" \
    --snapshot-name "$snapshot_id" \
    --region "$AWS_REGION" \
    --replication-group-description "Restored Redis for RHOAM" \
    --engine "$engine" \
    --engine-version "$engine_version" \
    --cache-subnet-group-name "$subnet_group" \
    --no-cli-pager

  aws elasticache wait replication-group-available --replication-group-id "$redis_id" --region "$AWS_REGION" --max-items 100 --no-cli-pager


  log "✅ Redis group $redis_id restored."
}


# --- SCRIPT START ---
log "Starting RHOAM Restore Process"

INTERACTIVE_MODE=false
echo "Please choose a run mode:"
echo " [B]atch: Run the entire script automatically from start to finish."
echo " [I]nteractive: Confirm before restoring each individual database."
echo
read -p "Enter run mode (B/I): " choice
case "$choice" in
  i|I ) INTERACTIVE_MODE=true; log "Running in INTERACTIVE mode.";;
  b|B )
    INTERACTIVE_MODE=false; log "Running in BATCH mode."
    echo "This script will delete and restore all components."
    for i in {5..1}; do echo -n "$i." && sleep 1; done && echo " Starting."
    ;;
  * ) echo "Invalid choice. Aborting."; exit 1;;
esac


# --- 1. PRE-RESTORE STEPS ---
log "STEP 1: PRE-RESTORE - Scaling down components and pausing operators"

log "Scaling down all application pods (Keycloak and 3scale)..."
# Scale down Keycloak
oc patch keycloak/rhsso -n redhat-rhoam-rhsso --type merge -p '{"spec":{"instances":0}}'
oc patch keycloak/rhssouser -n redhat-rhoam-user-sso --type merge -p '{"spec":{"instances":0}}'

# Scale down 3scale
oc scale deployment/threescale-operator-controller-manager-v2 --replicas=0 -n redhat-rhoam-3scale-operator
echo "Waiting for the 3scale operator to scale down..."
# oc wait deployment/threescale-operator-controller-manager-v2 --for=jsonpath='{.status.replicas}'=0 --timeout=120s -n redhat-rhoam-3scale-operator

for deployment in $(oc get deployment -n redhat-rhoam-3scale -o jsonpath='{.items[*].metadata.name}'); do
  oc scale deployment/"$deployment" --replicas=0 -n redhat-rhoam-3scale
done

log "Pausing Postgres and Redis operators (skipCreate:true)..."
oc patch postgres rhsso-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch postgres rhssouser-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch postgres threescale-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge

oc patch redis ratelimit-service-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch redis threescale-backend-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch redis threescale-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge

# --- 2. RESTORE RDS DATABASES ---
log "STEP 2: RESTORE - RDS Databases"
if [ "$INTERACTIVE_MODE" = true ]; then
    read -p "Do you want to restore the RHSSO database? (y/n): " rds_choice_1
    if [[ "$rds_choice_1" =~ ^[Yy]$ ]]; then
        restore_rds_instance "$RHSSO_DB_ID" "$RHSSO_DB_SNAPSHOT"
    else
        log "Skipping RHSSO RDS restore."
    fi

    read -p "Do you want to restore the User SSO database? (y/n): " rds_choice_2
    if [[ "$rds_choice_2" =~ ^[Yy]$ ]]; then
        restore_rds_instance "$USERSSO_DB_ID" "$USERSSO_DB_SNAPSHOT"
    else
        log "Skipping User SSO RDS restore."
    fi

    read -p "Do you want to restore the 3scale database? (y/n): " rds_choice_3
    if [[ "$rds_choice_3" =~ ^[Yy]$ ]]; then
        restore_rds_instance "$THREESCALE_DB_ID" "$THREESCALE_DB_SNAPSHOT"
    else
        log "Skipping 3scale RDS restore."
    fi
else
    restore_rds_instance "$RHSSO_DB_ID" "$RHSSO_DB_SNAPSHOT"
    restore_rds_instance "$USERSSO_DB_ID" "$USERSSO_DB_SNAPSHOT"
    restore_rds_instance "$THREESCALE_DB_ID" "$THREESCALE_DB_SNAPSHOT"
fi


# --- 3. RESTORE REDIS CLUSTERS ---
log "STEP 3: RESTORE - Redis (ElastiCache) Clusters"
if [ "$INTERACTIVE_MODE" = true ]; then
    read -p "Do you want to restore the Ratelimit Redis cluster? (y/n): " redis_choice_1
    if [[ "$redis_choice_1" =~ ^[Yy]$ ]]; then
        restore_redis_instance "$RATELIMIT_REDIS_ID" "$RATELIMIT_REDIS_SNAPSHOT"
    else
        log "Skipping Ratelimit Redis restore."
    fi

    read -p "Do you want to restore the Backend Redis cluster? (y/n): " redis_choice_2
    if [[ "$redis_choice_2" =~ ^[Yy]$ ]]; then
        restore_redis_instance "$BACKEND_REDIS_ID" "$BACKEND_REDIS_SNAPSHOT"
    else
        log "Skipping Backend Redis restore."
    fi

    read -p "Do you want to restore the System Redis cluster? (y/n): " redis_choice_3
    if [[ "$redis_choice_3" =~ ^[Yy]$ ]]; then
        restore_redis_instance "$SYSTEM_REDIS_ID" "$SYSTEM_REDIS_SNAPSHOT"
    else
        log "Skipping System Redis restore."
    fi
else
    restore_redis_instance "$RATELIMIT_REDIS_ID" "$RATELIMIT_REDIS_SNAPSHOT"
    restore_redis_instance "$BACKEND_REDIS_ID" "$BACKEND_REDIS_SNAPSHOT"
    restore_redis_instance "$SYSTEM_REDIS_ID" "$SYSTEM_REDIS_SNAPSHOT"
fi


# --- 4. RESTORE S3 BUCKET ---
log "STEP 4: RESTORE - S3 Bucket"
PROCEED_S3=true
if [ "$INTERACTIVE_MODE" = true ]; then
    read -p "Do you want to restore the S3 bucket? (y/n): " s3_choice
    if [[ ! "$s3_choice" =~ ^[Yy]$ ]]; then
        PROCEED_S3=false
    fi
fi

if [ "$PROCEED_S3" = true ]; then
    log "Syncing data from $S3_BACKUP_BUCKET to $S3_TARGET_BUCKET..."
    aws s3 sync s3://"$S3_BACKUP_BUCKET" s3://"$S3_TARGET_BUCKET" --delete --no-cli-pager
    log "✅ S3 sync complete."
else
    log "Skipping S3 restore."
fi


# --- 5. POST-RESTORE STEPS ---
log "STEP 5: POST-RESTORE - Unpausing operators and scaling up components"

log "Unpausing Postgres and Redis operators (skipCreate:false)..."
oc patch postgres rhsso-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
oc patch postgres rhssouser-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
oc patch postgres threescale-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
oc patch redis ratelimit-service-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
oc patch redis threescale-backend-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
oc patch redis threescale-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge

log "Scaling up Keycloak instances..."
oc patch keycloak/rhsso -n redhat-rhoam-rhsso --type merge -p '{"spec":{"instances":2}}'
oc patch keycloak/rhssouser -n redhat-rhoam-user-sso --type merge -p '{"spec":{"instances":2}}'

log "Scale up 3scale operator..."
oc scale deployment/threescale-operator-controller-manager-v2 --replicas=1 -n redhat-rhoam-3scale-operator
echo "Waiting for the 3scale operator to become available..."
oc wait deployment/threescale-operator-controller-manager-v2 --for=condition=Available=True --timeout=120s -n redhat-rhoam-3scale-operator

log "Restarting 3scale deployments..."
for deployment in $(oc get deployment -n redhat-rhoam-3scale -o jsonpath='{.items[*].metadata.name}'); do
  oc rollout restart deployment/"$deployment" -n redhat-rhoam-3scale
done

log "🎉 RHOAM RESTORE PROCESS COMPLETE! Please perform final verification checks."