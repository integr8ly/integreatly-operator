#!/bin/bash
#
# PREREQUISITES
# - jq
# - oc (logged in at the cmd line in order to get the bearer token)
# - a "perf-test-start-time.txt" file with a valid rfc3339 timestamp from a moment before performance tests started 
#   (it can be produced with this multiplatform command: `date -u +%Y-%m-%dT%TZ > perf-test-start-time.txt`)
# - a "perf-test-end-time.txt" file with a valid rfc3339 timestamp from a moment after performance tests finished
#
# VARIABLES
START_TIME_FILENAME=${START_TIME_FILENAME:-perf-test-start-time.txt}
END_TIME_FILENAME=${END_TIME_FILENAME:-perf-test-end-time.txt}
TOKEN=$(oc whoami --show-token)
#PROMETHEUS_ROUTE=$(echo "https://$(oc get route prometheus-route -n redhat-rhmi-middleware-monitoring-operator -o=jsonpath='{.spec.host}')")
PROMETHEUS_ROUTE=$(echo "https://$(oc get route prometheus-k8s -n openshift-monitoring -o=jsonpath='{.spec.host}')")
PROM_QUERY_ROUTE="$PROMETHEUS_ROUTE/api/v1/query"

# Get timestamps and calculate test duration
startTime=$(cat $START_TIME_FILENAME)
if [[ "$OSTYPE" == "darwin"* ]]; then
  # MacOS command:
  startTimestamp=$(date -u -j -f "%Y-%m-%dT%TZ" "$startTime" +"%s")
else
  # Linux command:
  startTimestamp=$(date -u -d "$startTime" +"%s")
fi

endTime=$(cat $END_TIME_FILENAME)
if [[ "$OSTYPE" == "darwin"* ]]; then
  # MacOS command:
  endTimestamp=$(date -u -j -f "%Y-%m-%dT%TZ" "$endTime" +"%s")
else
  # Linux command:
  endTimestamp=$(date -u -d "$endTime" +"%s")
fi
testDuration=$(($endTimestamp-$startTimestamp))

# Order of the queries must strictly match the rows from the spreadsheet that is used to collect these data
INSTANT_QUERIES=(\
  "sum(cluster:capacity_cpu_cores:sum)"\
  "sum(cluster:capacity_cpu_cores:sum{label_node_role_kubernetes_io!~'master|infra'})"\
  "sum(cluster:capacity_memory_bytes:sum)/1024/1024/1024"\
  "sum(cluster:capacity_memory_bytes:sum{label_node_role_kubernetes_io!~'master|infra'})/1024/1024/1024"\
)

# Order of the queries must strictly match the rows from the spreadsheet that is used to collect these data
IDLE_QUERIES=(\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-3scale', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-marin3r', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-user-sso', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-rhsso', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-middleware-monitoring-operator', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace=~'redhat-rhmi-.*', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-3scale'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-marin3r'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-user-sso'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-rhsso'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-middleware-monitoring-operator'} [15m])"\
  "sum(avg_over_time(namespace:container_cpu_usage:sum{namespace=~'redhat-rhmi-.*'} [15m]))"\
)

# Order of the queries must strictly match the rows from the spreadsheet that is used to collect these data
LOAD_QUERIES=(\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-3scale',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-marin3r',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-user-sso',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-rhsso',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhmi-middleware-monitoring-operator',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace=~'redhat-rhmi-.*',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-3scale'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-marin3r'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-user-sso'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-rhsso'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhmi-middleware-monitoring-operator'} [${testDuration}s])"\
  "sum(max_over_time(namespace:container_cpu_usage:sum{namespace=~'redhat-rhmi-.*'} [${testDuration}s]))"\
)

#
# Execute queries
#
for query in "${INSTANT_QUERIES[@]}";
do
  curl -s -G -H "Authorization: Bearer $TOKEN" --data-urlencode "query=$query" -H 'Accept: application/json' $PROM_QUERY_ROUTE | jq -r ".data.result[0].value[1]"
done

for query in "${IDLE_QUERIES[@]}";
do
  curl -s -G -H "Authorization: Bearer $TOKEN" --data-urlencode "query=$query" --data-urlencode "time=$startTime" -H 'Accept: application/json' $PROM_QUERY_ROUTE | jq -r ".data.result[0].value[1]"
done

for query in "${LOAD_QUERIES[@]}";
do
  curl -s -G -H "Authorization: Bearer $TOKEN" --data-urlencode "query=$query" --data-urlencode "time=$endTime" -H 'Accept: application/json' $PROM_QUERY_ROUTE | jq -r ".data.result[0].value[1]"
done