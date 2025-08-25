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


log() {
  echo
  echo "------------------------------------------------------------------"
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] - $1"
  echo "------------------------------------------------------------------"
}

pre_restore(){
  log "STEP 1: PRE-RESTORE - Scaling down components and pausing operators"

  log "Scaling down rhmi-operator deployment ..."
  oc scale deployment/rhmi-operator --replicas=0 -n redhat-rhoam-operator

  log "Scaling down Keycloak ..."
  oc patch keycloak/rhsso -n redhat-rhoam-rhsso --type merge -p '{"spec":{"instances":0}}'
  oc patch keycloak/rhssouser -n redhat-rhoam-user-sso --type merge -p '{"spec":{"instances":0}}'

  log "Pausing Postgres and Redis operators (skipCreate:true)..."
  oc patch postgres rhsso-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
  oc patch postgres rhssouser-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
  oc patch postgres threescale-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge

  oc patch redis ratelimit-service-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
  oc patch redis threescale-backend-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge
  oc patch redis threescale-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":true}}' --type=merge

  log "Scale Down 3scale Operator..."
  oc scale deployment/threescale-operator-controller-manager-v2 --replicas=0 -n redhat-rhoam-3scale-operator
  echo "Waiting 2 min for the 3scale operator to scale down..."
  oc wait deployment/threescale-operator-controller-manager-v2 --for=jsonpath='{.status.replicas}'=0 --timeout=120s -n redhat-rhoam-3scale-operator

  log "Scale Down 3scale..."
  for deployment in $(oc get deployment -n redhat-rhoam-3scale -o jsonpath='{.items[*].metadata.name}'); do
    oc scale deployment/"$deployment" --replicas=0 -n redhat-rhoam-3scale
  done

}

post_restore(){
  log "STEP 5: POST-RESTORE - Unpausing operators and scaling up components"

  oc scale deployment/rhmi-operator --replicas=1 -n redhat-rhoam-operator
  echo "Waiting for the rhmi-operator to become available..."
  oc wait deployment/rhmi-operator --for=condition=Available=True --timeout=60s


  log "Re-enable Redis CRs ..."
  oc patch redis ratelimit-service-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
  oc patch redis threescale-backend-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
  oc patch redis threescale-redis-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge

  log "Re-enable Postgres CRs ..."
  oc patch postgres rhsso-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
  oc patch postgres rhssouser-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge
  oc patch postgres threescale-postgres-rhoam -n redhat-rhoam-operator -p '{"spec":{"skipCreate":false}}' --type=merge

  log "Scale up and restart Keycloak..."
  oc patch keycloak/rhsso -n redhat-rhoam-rhsso --type merge -p '{"spec":{"instances":2}}'
  oc patch keycloak/rhssouser -n redhat-rhoam-user-sso --type merge -p '{"spec":{"instances":2}}'
  oc rollout restart statefulset/keycloak -n redhat-rhoam-rhsso
  oc rollout restart statefulset/keycloak -n redhat-rhoam-user-sso

  log "Scale up and restart 3scale..."
  oc scale deployment/threescale-operator-controller-manager-v2 --replicas=1 -n redhat-rhoam-3scale-operator
  echo "Waiting for the 3scale operator to become available..."
  oc wait deployment/threescale-operator-controller-manager-v2 --for=condition=Available=True --timeout=120s -n redhat-rhoam-3scale-operator

  log "Restarting 3scale deployments..."
  for deployment in $(oc get deployment -n redhat-rhoam-3scale -o jsonpath='{.items[*].metadata.name}'); do
    oc rollout restart deployment/"$deployment" -n redhat-rhoam-3scale
  done
}

update_secrets(){
# Get DB creds
  export DB_HOST=$(oc get secret threescale-postgres-rhoam -n redhat-rhoam-operator -o jsonpath='{.data.host}' | base64 --decode)
  export DB_USER=$(oc get secret threescale-postgres-rhoam -n redhat-rhoam-operator -o jsonpath='{.data.username}' | base64 --decode)
  export DB_PASSWORD=$(oc get secret threescale-postgres-rhoam -n redhat-rhoam-operator -o jsonpath='{.data.password}' | base64 --decode)
  export DB_NAME=$(oc get secret threescale-postgres-rhoam -n redhat-rhoam-operator -o jsonpath='{.data.database}' | base64 --decode)

  PSQL_CLIENT_POD=""
  while true; do
    echo "Select which pod to use for PostgreSQL client operations for Secrets updates:"
    echo "1) Use the existing 3scale system-app pod"
    echo "2) Create and use a temporary psql-client pod"
    read -p "Enter your choice (1 or 2): " choice

    case "$choice" in
      1)
        echo "Using the 3scale system-app pod."
        export PSQL_CLIENT_POD=$(oc get pod -n redhat-rhoam-3scale -l threescale_component=system,threescale_component_element=app -o jsonpath='{.items[0].metadata.name}')
        ;;
      2)
        # Check if the temporary pod already exists to avoid recreation
        if ! oc get pod psql-client -n redhat-rhoam-operator > /dev/null 2>&1; then
          echo "Creating temporary psql-client pod..."
          oc run psql-client --image=postgres:13 --restart=Never -n redhat-rhoam-operator -- /bin/sleep 3600
          oc wait --for=condition=Ready pod/psql-client -n redhat-rhoam-operator --timeout=300s
        else
            echo "Temporary psql-client pod already exists."
        fi
        export PSQL_CLIENT_POD="psql-client"
        echo "waiting for PSQL client pod to be ready..."
        oc wait --for=condition=Ready pod/$PSQL_CLIENT_POD -n redhat-rhoam-operator --timeout=300s
        ;;
      *)
        echo "Invalid choice. Please enter '1' or '2'."
        continue
        ;;
    esac


  if [ "$INTERACTIVE_MODE" = true ]; then
      read -p "Do you want to Update APICAST_TOKEN (y/n): " secret_choice_1
      if [[ "$secret_choice_1" =~ ^[Yy]$ ]]; then
        update_secrets_apicast_token
      else
          log "Skipping Update APICAST_TOKEN"
      fi

      read -p "Do you want to Update 3scale MASTER_ACCESS_TOKEN and MASTER_PASSWORD (y/n): " secret_choice_2
      if [[ "$secret_choice_2" =~ ^[Yy]$ ]]; then
        update_secrets_3scale_master_token_passwd
      else
          log "Skipping Update MASTER_ACCESS_TOKEN and MASTER_PASSWORD in system-seed secret."
      fi

      read -p "Do you want to update ADMIN_ACCESS_TOKEN and ADMIN_PASSWORD (y/n): " secret_choice_3
      if [[ "$secret_choice_3" =~ ^[Yy]$ ]]; then
          update_secrets_3scale_admin_token_passwd
      else
          log "Skipping UpdateADMIN_ACCESS_TOKEN and ADMIN_PASSWORD"
      fi

      read -p "Do you want to reset RHSSO and USER_SSO ADMIN_PASSWORD(s) (y/n): " secret_choice_4
      if [[ "$secret_choice_4" =~ ^[Yy]$ ]]; then
          update_secrets_kc_passwords
      else
          log "Skipping Update RHSSO and USER_SSO ADMIN_PASSWORDs"
      fi
  else
    update_secrets_apicast_token
    update_secrets_3scale_master_token_passwd
    update_secrets_3scale_admin_token_passwd
    update_secrets_kc_passwords
  fi
  done
}

update_secrets_apicast_token(){
  echo "Retrieving the Apicast access token..."
  APICAST_TOKEN=$(oc rsh -n redhat-rhoam-3scale "$PSQL_CLIENT_POD" \
    psql "host=$DB_HOST user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME" \
    --tuples-only --no-align \
    -c "SELECT value FROM access_tokens WHERE name = 'APIcast';" | tr -d '[:space:]')

  oc patch secret system-master-apicast -n redhat-rhoam-operator \
    --type='json' \
    -p="[{\"op\": \"replace\", \"path\": \"/data/ACCESS_TOKEN\", \"value\": \"$(echo -n $APICAST_TOKEN | base64)\"}]"

  echo "Apicast secret has been synced."
}

update_secrets_3scale_master_token_passwd(){

  MASTER_TOKEN=$(oc rsh -n redhat-rhoam-3scale system-app-5f6c5565f7-6f96p \
  sh -c "env PSQL_PAGER='' psql 'host=vmoccs4qzw65redhatrhoamoperatorthre-5grc.ciscu7upsfrm.us-east-2.rds.amazonaws.com \
  user=postgres password=e813fec85cf2464c833e38c34bcf3f6a dbname=postgres'
  --tuples-only --no-align -c 'SELECT value FROM access_tokens WHERE owner_id = 1 AND length(value) = 8 ORDER BY id DESC LIMIT 1;' | tr -d '\n' ")

  oc patch secret system-seed -n redhat-rhoam-operator \
    --type='json' \
    -p="[{\"op\": \"replace\", \"path\": \"/data/MASTER_ACCESS_TOKEN\", \"value\": \"$(echo -n $MASTER_TOKEN | base64)\"}]"

  MASTER_ROUTE=$(oc get route -n redhat-rhoam-3scale -o json | jq -r '.items[] | select(.spec.host | test("master")) |.spec.host')

  MASTER_PASSWORD=$(oc get secret system-seed -n redhat-rhoam-operator -o jsonpath='{.data.MASTER_PASSWORD}' | base64 --decode)

  curl -X PUT \
   "https://${MASTER_ROUTE}/admin/api/users/1.xml" \
   -H 'accept: */*' \
   -H 'Content-Type: application/x-www-form-urlencoded' \
   -d "access_token=${MASTER_TOKEN}&password=${MASTER_PASSWORD}"

    echo "Master password has been reset in the database."
}

update_secrets_3scale_admin_token_passwd(){
  ADMIN_TOKEN=$(oc rsh -n redhat-rhoam-3scale "$PSQL_CLIENT_POD" \
   psql "host=$DB_HOST user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME" \
   --tuples-only --no-align \
   -c "SELECT value FROM access_tokens WHERE owner_id = 2" \
   | tr -d '\n')

  oc patch secret system-seed -n redhat-rhoam-operator \
    --type='json' \
    -p="[{\"op\": \"replace\", \"path\": \"/data/ADMIN_ACCESS_TOKEN\", \"value\": \"$(echo -n $ADMIN_TOKEN | base64)\"}]"

  ADMIN_ROUTE=$(oc get route -n redhat-rhoam-3scale -o json | jq -r '.items[] | select(.spec.host | test("admin")) |.spec.host')

  ADMIN_PASSWORD=$(oc get secret system-seed -n redhat-rhoam-operator -o jsonpath='{.data.ADMIN_PASSWORD}' | base64 --decode)

  curl -X 'PUT' \
    "https://${ADMIN_ROUTE}/admin/api/accounts/2/users/2.xml" \
    -H 'accept: */*' \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -d "access_token=${ADMIN_TOKEN}&password=${ADMIN_PASSWORD}"

  echo "3scale admin credentials have been reset and synced."
}

update_secrets_kc_passwords() {
  oc scale deployment/rhsso-operator --replicas=0 -n redhat-rhoam-rhsso-operator
  oc scale statefulsets/keycloak --replicas=0 -n redhat-rhoam-rhsso

  NEW_ADMIN_USERNAME_B64=$(echo -n "new-admin" | base64)
  NEW_ADMIN_PASSWORD_B64=$(openssl rand -hex 8 | xargs echo -n | base64)

  oc patch secret credential-rhsso -n redhat-rhoam-operator \
   --type='json' \
   -p="[{\"op\": \"add\", \"path\": \"/data/ADMIN_USERNAME\", \"value\": \"$NEW_ADMIN_USERNAME_B64\"}, {\"op\": \"add\", \"path\": \"/data/ADMIN_PASSWORD\", \"value\": \"$NEW_ADMIN_PASSWORD_B64\"}]"

  oc scale statefulset/keycloak --replicas=2 -n redhat-rhoam-rhsso
  oc wait --for=condition=ready statefulset/keycloak -n redhat-rhoam-rhsso --timeout=120s

  oc scale deployment/rhsso-operator --replicas=0 -n redhat-rhoam-rhsso-operator

  echo "Checking Keycloak logs for new user creation..."
  oc logs keycloak-0 -n redhat-rhoam-rhsso | grep "Added user 'new-admin'"

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
  local security_group=$(echo "$snapshot_info" | jq -r '.NodeSnapshots[0].VpcSecurityGroupId')

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

  if [[ -z "$security_group" || "$security_group" == "null" ]]; then
    echo; log "⚠️ WARNING: Could not automatically determine the Security Group from the snapshot."
    read -p "Enter the Security Group ID: " security_group
    if [[ -z "$security_group" ]]; then echo "Aborting."; exit 1; fi
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
    --num-cache-clusters 2 \
    --snapshot-retention-limit 30 \
    --automatic-failover-enabled \
    --engine "$engine" \
    --engine-version "$engine_version" \
    --cache-subnet-group-name "$subnet_group" \
    --security-group-ids  "$security_group" \
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
log "STEP 1: PRE-RESTORE STEPS"
if [ "$INTERACTIVE_MODE" = true ]; then
    read -p "Do you want to run PRE-RESTORE STEPS (y/n): " pre_choice_1
    if [[ "$pre_choice_1" =~ ^[Yy]$ ]]; then
        pre_restore
    else
        log "Skipping PRE-RESTORE STEPS"
    fi
fi

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
if [ "$INTERACTIVE_MODE" = true ]; then
    read -p "Do you want to run Post_restore? (y/n): " post_restore_1
    if [[ "$post_restore_1" =~ ^[Yy]$ ]]; then
        post_restore
    else
        log "Skipping post_restore"
    fi
fi


if [ "$INTERACTIVE_MODE" = true ]; then
    read -p "Do you want to Update secrets? (y/n): " update_secrets_1
    if [[ "$update_secrets_1" =~ ^[Yy]$ ]]; then
        update_secrets
    else
        log "Skipping update secrets"
    fi
fi


log "🎉 RHOAM RESTORE PROCESS COMPLETE! Please perform final verification checks."