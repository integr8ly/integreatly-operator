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
#PROMETHEUS_ROUTE=$(echo "https://$(oc get route prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o=jsonpath='{.spec.host}')")
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
  "sum(kube_node_status_allocatable_cpu_cores * on (node) (kube_node_role{role='worker'} == on (node) group_left () (count by (node) (kube_node_role{}))))"\
  "sum(cluster:capacity_memory_bytes:sum)/1024/1024/1024"\
  "sum(cluster:capacity_memory_bytes:sum{label_node_role_kubernetes_io!~'master|infra'})/1024/1024/1024"\
  "kube_node_status_allocatable_memory_bytes * on (node) (kube_node_role{role='worker'} == on (node) group_left () (count by (node) (kube_node_role{}))) / 1024 / 1024 / 1024"\
  "sum(kube_pod_container_resource_requests_cpu_cores{namespace=~'redhat-rhoam-.*',container!='lifecycle'} * on(namespace, pod) group_left() max by (namespace, pod) ( kube_pod_status_phase{phase='Running'} == 1 ))"\
  "sum(kube_pod_container_resource_requests_memory_bytes{namespace=~'redhat-rhoam-.*', container!='lifecycle'} * on(namespace, pod) group_left() max by (namespace, pod) ( kube_pod_status_phase{phase='Running'} == 1 )) / 1024 /1024"\
)

# Order of the queries must strictly match the rows from the spreadsheet that is used to collect these data
IDLE_QUERIES=(\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-3scale', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-marin3r', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-user-sso', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-rhsso', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-middleware-monitoring-operator', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(sum(container_memory_working_set_bytes{namespace=~'redhat-rhoam-.*', pod!='', container=''}) [15m:10s])/1024/1024"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-3scale'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-marin3r'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-user-sso'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-rhsso'} [15m])"\
  "avg_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-middleware-monitoring-operator'} [15m])"\
  "sum(avg_over_time(namespace:container_cpu_usage:sum{namespace=~'redhat-rhoam-.*'} [15m]))"\
)

# Order of the queries must strictly match the rows from the spreadsheet that is used to collect these data
LOAD_QUERIES=(\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-3scale',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-marin3r',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-user-sso',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-rhsso',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace='redhat-rhoam-middleware-monitoring-operator',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(sum(container_memory_working_set_bytes{namespace=~'redhat-rhoam-.*',container='', pod!=''}) [${testDuration}s:10s])/1024/1024"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-3scale'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-marin3r'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-user-sso'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-rhsso'} [${testDuration}s])"\
  "max_over_time(namespace:container_cpu_usage:sum{namespace='redhat-rhoam-middleware-monitoring-operator'} [${testDuration}s])"\
  "sum(max_over_time(namespace:container_cpu_usage:sum{namespace=~'redhat-rhoam-.*'} [${testDuration}s]))"\
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