#!/bin/sh

test_redis_backup () {
  REDIS_CR_NAME=$1
  AWS_REDIS_ID=$2

  # Get the Redis host credentials
  REDIS_HOST=`oc get secrets/$REDIS_CR_NAME -n redhat-rhmi-operator -o template --template={{.data.uri}} | base64 -d`

  # Disable the AWS pager to avoid the user to be displayed with an interactive
  # output when using the AWS CLI
  AWS_PAGER=""

  # Pull Redis throw-away container name
  POD_NAME=$(oc get pod -n redhat-rhmi-operator | grep -v "deploy" | grep throw-away-redis-pod | awk '{print $1}')

  # Wait for Redis throwaway to be in running state
  while true
  do
      PHASE=`oc get pod $POD_NAME -n redhat-rhmi-operator -o template --template={{.status.phase}}`
      echo $POD_NAME
      if [ "$PHASE" = 'Running' ]; then
        break
      fi

      echo "Waiting for Redis instance to be created, current phase is: $PHASE..."
      sleep 10s
  done

  # Create a test key in redis cache before taking a snapshot
  create_redis_key

  # Create the snapshot
  REDIS_SNAPSHOT_NAME="${REDIS_CR_NAME}-snapshot-test-$(date +"%Y-%m-%d-%H%M%S")"
  echo "Creating snapshot $REDIS_SNAPSHOT_NAME..."
    cat << EOF | oc create -f - -n redhat-rhmi-operator
    apiVersion: integreatly.org/v1alpha1
    kind: RedisSnapshot
    metadata:
        # Needs to be a unique name for this snapshot
        name: $REDIS_SNAPSHOT_NAME
    spec:
        # The Redis resource name for the snapshot you want to take
        resourceName: $REDIS_CR_NAME
EOF

  # Wait for it to complete
  while true
  do
      PHASE=`oc get redissnapshot/$REDIS_SNAPSHOT_NAME -n redhat-rhmi-operator -o template --template={{.status.phase}}`
      if [ "$PHASE" = 'complete' ]; then
        echo "Snapshot creation completed."  
        break
      fi

      echo "Waiting for snapshot to complete. Current phase: $PHASE..."
      sleep 10s
  done

  # Edit Redis CR to prevent recreation during restoration
  echo "Disabling automatic RDS recreation..."
  oc patch redis/$REDIS_CR_NAME -n redhat-rhmi-operator -p '{"spec":{"skipCreate":true}}' --type merge  

  # Get VPC security group IDs from existing Elasticache
  SECURITY_GROUP_IDS=$(aws elasticache describe-cache-clusters | jq --arg AWS_REDIS_ID "$AWS_REDIS_ID" '.CacheClusters | map(select(.ReplicationGroupId == $AWS_REDIS_ID))[0].SecurityGroups[].SecurityGroupId' -r | tr '\n' ' ' | sed -e 's/[[:space:]]$//')
  echo "Obtained VPC Security Group IDs: $SECURITY_GROUP_IDS"

  # Get Subnet group name from existing Elasticache
  CACHE_SUBNET_GROUP_NAME=$(aws elasticache describe-cache-clusters | jq --arg AWS_REDIS_ID "$AWS_REDIS_ID" '.CacheClusters | map(select(.ReplicationGroupId == $AWS_REDIS_ID))[0].CacheSubnetGroupName' -r)
  echo "Obtained Subnet group name: $CACHE_SUBNET_GROUP_NAME"

  # Delete replication group
  echo "Deleting replication group..."
  aws elasticache delete-replication-group \
	--replication-group-id $AWS_REDIS_ID 

  echo "Describe status of redis cache cluster"
  while true
  do
    # Check if the redis replication group exists
    EXISTS=`aws elasticache describe-replication-groups --replication-group-id $AWS_REDIS_ID`
    if [ "$EXISTS" = 'false' ]; then
      echo "Redis cache deleted"
      break
    fi

    # Attempt to get the redis cache status. If it fails, check if the error is
    # not found. If it's not found it means that redis was deleted, so break
    # the loop. Otherwise report the error
    REDIS_CACHE=`aws elasticache describe-replication-groups --replication-group-id $AWS_REDIS_ID 2>&1`
    if [ ! "$?" = 0 ]; then
      if echo $REDIS_CACHE | grep -q "ReplicationGroupNotFoundFault"; then
        echo "Redis Cache deleted"
        break
      fi

      echo "Unexpected error requesting redis cache: $REDIS_CACHE"
      exit 1
    fi

    # Assert that, as the redis hasn't been deleted yet
    STATUS=`echo $REDIS_CACHE | jq -r '.ReplicationGroups[0].Status'`
    if [ "$STATUS" = 'deleting' ]; then
      echo "Waiting for redis cache deletion..."
      sleep 10s
      continue
    fi

    # If the Redis still exists but the status is not deleting, fail the test
    echo "Unexpected status '$STATUS' when deleting redis cache"
    exit 1
  done

  # Restore the database
  echo "Restoring redis from snapshot..."

  REDIS_RESTORE_SNAPSHOT=`oc get redissnapshots $REDIS_SNAPSHOT_NAME -n redhat-rhmi-operator -o json | jq -r '.status.snapshotID'`

  aws elasticache create-replication-group \
	 --replication-group-id $AWS_REDIS_ID \
	 --replication-group-description "A Redis replication group" \
	 --num-cache-clusters 2 \
	 --snapshot-retention-limit 30 \
	 --automatic-failover-enabled \
	 --cache-subnet-group-name $CACHE_SUBNET_GROUP_NAME \
	 --security-group-ids $SECURITY_GROUP_IDS \
	 --snapshot-name $REDIS_RESTORE_SNAPSHOT

  # Wait for redis to be available
  while true
  do
    STATUS=$(aws elasticache describe-replication-groups --replication-group-id $AWS_REDIS_ID | jq -r '.ReplicationGroups[0].Status')
    if [ "$STATUS" = 'available' ]; then
      echo "Redis restored."
      break
    fi
    echo "Waiting for snapshot restoration... Current status: $STATUS"
    sleep 10s
  done

  # Revert Redis CR Change
  echo "Re-enabling automating Redis recreation..."
  oc patch redis/$REDIS_CR_NAME -n redhat-rhmi-operator -p '{"spec":{"skipCreate":false}}' --type merge
}

# Create throwaway Redis container
create_redis_pod() {
  echo "Creating Redis container..."

  cat << EOF | oc create -f - -n redhat-rhmi-operator
  apiVersion: integreatly.org/v1alpha1
  kind: Redis
  metadata:
    name: throw-away-redis-pod
    labels:
      productName: productName
  spec:
    secretRef:
      name: throw-away-redis-sec
    tier: development
    type: workshop
EOF
}

# Create a test key in Redis Cache
create_redis_key(){
  echo "Creating a test entry in Redis Cache"

  kubectl exec $POD_NAME \
    -n redhat-rhmi-operator \
    -- /opt/rh/rh-redis32/root/usr/bin/redis-cli -c -h $REDIS_HOST -p 6379 SET mykey "Test key"
}

# Check if the test key created is available after recovery
get_redis_key(){
  POD_NAME=$(oc get pod -n redhat-rhmi-operator | grep -v "deploy" | grep throw-away-redis-pod | awk '{print $1}')
  REDIS_CR_NAME=$1
  REDIS_HOST=$(oc get secrets/$REDIS_CR_NAME -n redhat-rhmi-operator -o template --template={{.data.uri}} | base64 -d)

  # Get the key after backup and restore
  echo "Reading test entry from Redis Cache"

  RESULT=$(kubectl exec $POD_NAME \
    -n redhat-rhmi-operator \
    -- /opt/rh/rh-redis32/root/usr/bin/redis-cli -c -h $REDIS_HOST -p 6379 GET mykey)

  if [ ! "$RESULT" = "Test key" ]; then
      echo "Test failed, test key doesn't exist in restored Redis cache"
      exit 1
  fi
}

delete_throw_away_pod(){
  echo "Deleting throw-away-pod"
  # Delete the redis pod
  oc delete redis/throw-away-redis-pod -n redhat-rhmi-operator
}
