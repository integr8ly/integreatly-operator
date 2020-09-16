#!/bin/sh

# Import the test function
. ./postgres.sh --source-only
. ./redis.sh --source-only
. ./system-app.sh --source-only

# Set the parameters
AWS_DB_ID=$(oc get secret/system-database -o go-template --template="{{.data.URL|base64decode}}" -n redhat-rhmi-3scale | grep -Po "(?<=@).*?(?=\.)")
POSTGRES_CR_NAME=threescale-postgres-rhmi
DATABASE_SECRET=threescale-postgres-rhmi

AWS_BACKEND_REDIS_ID=$(oc get secrets/backend-redis -o go-template --template="{{.data.REDIS_QUEUES_URL|base64decode}}" -n redhat-rhmi-3scale | grep -Po "(?<=/\/).*?(?=\.)")
BACKEND_REDIS_CR_NAME=threescale-backend-redis-rhmi

AWS_SYSTEM_REDIS_ID=$(oc get secrets/system-redis -o go-template --template="{{.data.MESSAGE_BUS_URL|base64decode}}" -n redhat-rhmi-3scale | grep -Po "(?<=/\/).*?(?=\.)")
SYSTEM_REDIS_CR_NAME=threescale-redis-rhmi

echo "Testing Redis Postgres database backup and restore"
test_postgres_backup $POSTGRES_CR_NAME $DATABASE_SECRET $AWS_DB_ID

echo ""
echo "Creating Redis throw-away pod"
create_redis_pod

echo ""
echo "Testing Redis System backup and restore"
test_redis_backup $SYSTEM_REDIS_CR_NAME $AWS_SYSTEM_REDIS_ID
echo "Testing Redis System Backend backup and restore was successful"

echo ""
echo "Testing Redis Backend backup and restore"
test_redis_backup $BACKEND_REDIS_CR_NAME $AWS_BACKEND_REDIS_ID
echo "Testing Redis Backend backup and restore was successful"

echo ""
echo "Testing System App"
test_system_app
echo "All tests passed successfully"

echo ""
echo "Checking if backend Redis test key exists..."
get_redis_key $BACKEND_REDIS_CR_NAME
echo "Testing Redis backend was successful"

echo ""
echo "Checking if backend Redis test key exists..."
get_redis_key $SYSTEM_REDIS_CR_NAME
echo "Testing Redis system was successful"

echo ""
echo "Deleting Redis throw-away pod"
delete_throw_away_pod

echo ""
echo "All tests run successfully"