#!/bin/bash

# ==============================================================================
# RHOAM ADDON - AUTOMATED DISASTER RECOVERY RESTORE SCRIPT
# ==============================================================================
# This script automates the full restore process for a RHOAM AddOn installation.
# It reads all necessary parameters from the 'restore_rhoam_addon_config.sh' file.
# ==============================================================================

set -e
set -o pipefail

# --- LOAD CONFIGURATION ---
source ./restore_rhoam_addon_config.sh
# --- END OF CONFIGURATION ---


# --- HELPER FUNCTIONS ---

log() {
  echo
  echo "------------------------------------------------------------------"
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] - $1"
  echo "------------------------------------------------------------------"
}

# --- SCRIPT START ---

log "Starting RHOAM Restore Process"
echo "This script will delete and restore 3 RDS databases, 3 Redis clusters, and sync an S3 bucket."
echo "Please review the configuration above before proceeding."
for i in {5..1}; do echo -n "$i." && sleep 1; done && echo " Starting."


# --- 1. PRE-RESTORE STEPS ---
log "STEP 1: PRE-RESTORE - Scaling down components and pausing operators"

log "Scaling down Keycloak instances..."
oc patch keycloak/rhsso -n redhat-rhoam-rhsso --type merge -p '{"spec":{"instances":0}}'
oc patch keycloak/rhssouser -n redhat-rhoam-user-sso --type merge -p '{"spec":{"instances":0}}'

log "Pausing Postgres and Redis operators (skipCreate:true)..."
oc patch postgres rhsso-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch postgres rhssouser-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch postgres threescale-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge

oc patch redis ratelimit-service-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch redis threescale-backend-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
oc patch redis threescale-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge


# --- 2. RESTORE RDS DATABASES ---
log "STEP 2: RESTORE - RDS Databases"

restore_rds_instance() {
  local db_id=$1
  local snapshot_id=$2
  log "Processing RDS Instance: $db_id"

  log "Capturing config from snapshot $snapshot_id..."
  local snapshot_info=$(aws rds describe-db-snapshots --db-snapshot-identifier "$snapshot_id" --region "$AWS_REGION" --query "DBSnapshots[0]")
  local instance_class=$(echo "$snapshot_info" | jq -r '.DBInstanceClass')
  local subnet_group=$(echo "$snapshot_info" | jq -r '.DBSubnetGroupName')
  local vpc_sgs=$(echo "$snapshot_info" | jq -r '.VpcSecurityGroupIds | join(" ")')

  # Check for null Instance Class
  if [[ -z "$instance_class" || "$instance_class" == "null" ]]; then
    echo
    log "⚠️ WARNING: Could not automatically determine the DB Instance Class from the snapshot."
    read -p "Enter DB Instance Class (e.g., db.m5.large): " instance_class
    if [[ -z "$instance_class" ]]; then
      echo "No instance class provided. Aborting."
      exit 1
    fi
    log "Using manually provided instance class: $instance_class"
  fi

  # Check for null Subnet Group
  if [[ -z "$subnet_group" || "$subnet_group" == "null" ]]; then
    echo
    log "⚠️ WARNING: Could not automatically determine the DB Subnet Group from the snapshot."
    read -p "Enter DB Subnet Group Name: " subnet_group
    if [[ -z "$subnet_group" ]]; then
      echo "No subnet group provided. Aborting."
      exit 1
    fi
    log "Using manually provided subnet group: $subnet_group"
  fi

  # Check for empty Security Groups
  if [[ -z "$vpc_sgs" ]]; then
    echo
    log "⚠️ WARNING: Could not automatically determine the VPC Security Groups from the snapshot."
    echo "Please provide a valid, space-separated list of Security Group IDs."
    echo "(To list SGs in your VPC, run this in another terminal:"
    echo "aws ec2 describe-security-groups --region $AWS_REGION --filters Name=vpc-id,Values=<your-vpc-id> --query 'SecurityGroups[*].GroupId' --output text)"
    echo
    read -p "Enter VPC Security Group IDs: " vpc_sgs
    if [[ -z "$vpc_sgs" ]]; then
        echo "No security groups provided. Aborting."
        exit 1
    fi
    log "Using manually provided security groups: $vpc_sgs"
  fi

  log "Checking existence of RDS instance $db_id..."
  if aws rds describe-db-instances --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager > /dev/null 2>&1; then
    log "Instance found. Proceeding with deletion..."
    aws rds modify-db-instance --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-deletion-protection --apply-immediately --no-cli-pager
    aws rds delete-db-instance --db-instance-identifier "$db_id" --region "$AWS_REGION" --skip-final-snapshot --no-cli-pager
    aws rds wait db-instance-deleted --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager
  else
    echo
    log "⚠️ WARNING: RDS instance '$db_id' not found."
    read -p "Do you want to [S]kip deletion and proceed with restore, or [A]bort? (S/A): " choice
    case "$choice" in
      s|S ) echo "Skipping deletion, proceeding with restore...";;
      a|A ) echo "Aborting script."; exit 1;;
      * ) echo "Invalid choice. Aborting."; exit 1;;
    esac
  fi

  log "Restoring $db_id from snapshot $snapshot_id..."
  aws rds restore-db-instance-from-db-snapshot \
    --db-instance-identifier "$db_id" \
    --db-snapshot-identifier "$snapshot_id" \
    --region "$AWS_REGION" \
    --db-instance-class "$instance_class" \
    --db-subnet-group-name "$subnet_group" \
    --vpc-security-group-ids "$vpc_sgs" \
    --no-multi-az \
    --deletion-protection \
    --no-cli-pager
  aws rds wait db-instance-available --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager
  log "✅ RDS instance $db_id restored."
}


restore_rds_instance "$RHSSO_DB_ID" "$RHSSO_DB_SNAPSHOT"
restore_rds_instance "$USERSSO_DB_ID" "$USERSSO_DB_SNAPSHOT"
restore_rds_instance "$THREESCALE_DB_ID" "$THREESCALE_DB_SNAPSHOT"


# --- 3. RESTORE REDIS CLUSTERS ---
log "STEP 3: RESTORE - Redis (ElastiCache) Clusters"
restore_rds_instance() {
  local db_id=$1
  local snapshot_id=$2
  log "Processing RDS Instance: $db_id"

  log "Capturing config from snapshot $snapshot_id..."
  local snapshot_info=$(aws rds describe-db-snapshots --db-snapshot-identifier "$snapshot_id" --region "$AWS_REGION" --query "DBSnapshots[0]")
  local instance_class=$(echo "$snapshot_info" | jq -r '.DBInstanceClass')
  local subnet_group=$(echo "$snapshot_info" | jq -r '.DBSubnetGroupName')
  local vpc_sgs=$(echo "$snapshot_info" | jq -r '.VpcSecurityGroupIds | join(" ")')

  # Check for null Instance Class
  if [[ -z "$instance_class" || "$instance_class" == "null" ]]; then
    echo
    log "⚠️ WARNING: Could not automatically determine the DB Instance Class from the snapshot."
    read -p "Enter DB Instance Class (e.g., db.m5.large): " instance_class
    if [[ -z "$instance_class" ]]; then
      echo "No instance class provided. Aborting."
      exit 1
    fi
    log "Using manually provided instance class: $instance_class"
  fi

  # Check for null Subnet Group
  if [[ -z "$subnet_group" || "$subnet_group" == "null" ]]; then
    echo
    log "⚠️ WARNING: Could not automatically determine the DB Subnet Group from the snapshot."
    read -p "Enter DB Subnet Group Name: " subnet_group
    if [[ -z "$subnet_group" ]]; then
      echo "No subnet group provided. Aborting."
      exit 1
    fi
    log "Using manually provided subnet group: $subnet_group"
  fi

  log "⚠️ WARNING: Check for empty Security Groups"
  # Check for empty Security Groups
  if [[ -z "$vpc_sgs" ]]; then
    echo
    log "⚠️ WARNING: Could not automatically determine the VPC Security Groups from the snapshot."
    echo "Please provide a valid, space-separated list of Security Group IDs."
    echo "(To list SGs in your VPC, run this in another terminal:"
    echo "aws ec2 describe-security-groups --region $AWS_REGION --filters Name=vpc-id,Values=<your-vpc-id> --query 'SecurityGroups[*].GroupId' --output text)"
    echo
    read -p "Enter VPC Security Group IDs: " vpc_sgs
    if [[ -z "$vpc_sgs" ]]; then
        echo "No security groups provided. Aborting."
        exit 1
    fi
    log "Using manually provided security groups: $vpc_sgs"
  fi

  log "Checking existence of RDS instance $db_id..."
  if aws rds describe-db-instances --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager > /dev/null 2>&1; then
    log "Instance found. Proceeding with deletion..."
    aws rds modify-db-instance --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-deletion-protection --apply-immediately --no-cli-pager
    aws rds delete-db-instance --db-instance-identifier "$db_id" --region "$AWS_REGION" --skip-final-snapshot --no-cli-pager
    aws rds wait db-instance-deleted --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager
  else
    echo
    log "⚠️ WARNING: RDS instance '$db_id' not found."
    read -p "Do you want to [S]kip deletion and proceed with restore, or [A]bort? (S/A): " choice
    case "$choice" in
      s|S ) echo "Skipping deletion, proceeding with restore...";;
      a|A ) echo "Aborting script."; exit 1;;
      * ) echo "Invalid choice. Aborting."; exit 1;;
    esac
  fi

  log "Restoring $db_id from snapshot $snapshot_id..."
  aws rds restore-db-instance-from-db-snapshot \
    --db-instance-identifier "$db_id" \
    --db-snapshot-identifier "$snapshot_id" \
    --region "$AWS_REGION" \
    --db-instance-class "$instance_class" \
    --db-subnet-group-name "$subnet_group" \
    --vpc-security-group-ids "$vpc_sgs" \
    --no-multi-az \
    --deletion-protection \
    --no-cli-pager
  aws rds wait db-instance-available --db-instance-identifier "$db_id" --region "$AWS_REGION" --no-cli-pager
  log "✅ RDS instance $db_id restored."
}

restore_redis_instance "$RATELIMIT_REDIS_ID" "$RATELIMIT_REDIS_SNAPSHOT"
restore_redis_instance "$BACKEND_REDIS_ID" "$BACKEND_REDIS_SNAPSHOT"
restore_redis_instance "$SYSTEM_REDIS_ID" "$SYSTEM_REDIS_SNAPSHOT"


# --- 4. RESTORE S3 BUCKET ---
log "STEP 4: RESTORE - S3 Bucket"

log "Syncing data from $S3_BACKUP_BUCKET to $S3_TARGET_BUCKET..."
# Uncomment the next line to perform a dry run first
# aws s3 sync --dryrun s3://"$S3_BACKUP_BUCKET" s3://"$S3_TARGET_BUCKET" --no-cli-pager
aws s3 sync s3://"$S3_BACKUP_BUCKET" s3://"$S3_TARGET_BUCKET" --delete --no-cli-pager
log "✅ S3 sync complete."


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

log "Restarting 3scale deployments..."
oc rollout restart deployment/apicast-production -n redhat-rhoam-3scale
oc rollout restart deployment/apicast-staging -n redhat-rhoam-3scale
oc rollout restart deployment/backend-listener -n redhat-rhoam-3scale
oc rollout restart deployment/system-app -n redhat-rhoam-3scale
oc rollout restart deployment/system-sidekiq -n redhat-rhoam-3scale
oc rollout restart deployment/zync-que -n redhat-rhoam-3scale
oc rollout restart deployment/zync -n redhat-rhoam-3scale

log "🎉 RHOAM RESTORE PROCESS COMPLETE! Please perform final verification checks."