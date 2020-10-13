#!/bin/bash
# USAGE
# ./alerts-during-perf-testing.sh <optional product-name>
# ^C to break
# Generates two files one for firing and one for pending alerts
#
# PREREQUISITES
# - jq
# - oc (logged in at the cmd line in order to get the bearer token)
# VARIABLES
TOKEN=$(oc whoami --show-token)
MONITORING_ROUTE=$(echo "https://$(oc get route prometheus-route -n redhat-rhmi-middleware-monitoring-operator -o=jsonpath='{.spec.host}')")/api/v1/alerts

# remove tmp files on ctrl-c
trap "rm tmp-alert-firing-during-perf-testing-report.csv tmp-alert-pending-during-perf-testing-report.csv" EXIT

# function to check if there are no alerts firing bar deadmansnitch
function CHECK_NO_ALERTS(){
    if [[ $(curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE | jq -r '.data.alerts[]| select(.state=="firing") | [.labels.alertname, .state, .activeAt ] | @csv' | wc -l) == 1 ]] ; then
      echo Only alert firing is DeadMansSwitch
      date
    fi
}

# If no args are passed in then run for all products
if (( $# == 0 )); then
  while :; do
    # generate a report
    curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
    | jq -r '.data.alerts[]| select(.state=="pending") | [.labels.alertname, .state, .activeAt ] | @csv'>> tmp-alert-pending-during-perf-testing-report.csv

    curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
    | jq -r '.data.alerts[]| select(.state=="firing") | [.labels.alertname, .state, .activeAt ] | @csv'>> tmp-alert-firing-during-perf-testing-report.csv

    # sort command to remove duplicate alert
    sort -t',' -k 1,1 -u tmp-alert-firing-during-perf-testing-report.csv > alert-firing-during-perf-testing-report.csv
    #sort -t, -k1 -u alert-firing-during-perf-testing-report.csv > alert-firing-during-perf-testing-report.csv
    sort -t, -k1 -u tmp-alert-pending-during-perf-testing-report.csv > alert-pending-during-perf-testing-report.csv
    CHECK_NO_ALERTS
  done
fi

# Check the alerts in product namespaces
PRODUCT_NAME=`echo $1 | grep "3scale\|user-sso\|rhsso\|marin3r"`

if [ -z "$PRODUCT_NAME" ]; then
  echo Error command line either no args or args excepted for individual products alerts listed below
  echo - "3scale"
  echo - "user-sso"
  echo - "rhsso"
  echo - "marin3r"
else
  echo "Check $1 product namespace and operator namespace for alerts"
  while :; do
    curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
    | jq -r '.data.alerts[]| select(.state=="pending" or .labels.namespace=="redhat-rhmi-'$PRODUCT_NAME'" or .labels.namespace=="redhat-rhmi-'$PRODUCT_NAME'-operator") | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-pending-during-perf-testing-report.csv

    curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
    | jq -r '.data.alerts[]| select(.state=="firing" or .labels.namespace=="redhat-rhmi-'$PRODUCT_NAME'" or .labels.namespace=="redhat-rhmi-'$PRODUCT_NAME'-operator") | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-firing-during-perf-testing-report.csv

    # sort command to remove duplicate alert
    sort -t, -k1 -u tmp-alert-pending-during-perf-testing-report.csv > ${PRODUCT_NAME}-alert-pending-during-perf-testing-report.csv
    sort -t, -k1 -u tmp-alert-firing-during-perf-testing-report.csv > ${PRODUCT_NAME}-alert-firing-during-perf-testing-report.csv

    CHECK_NO_ALERTS
  done
fi

