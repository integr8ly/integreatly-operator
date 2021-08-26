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
NAMESPACE_PREFIX="${NAMESPACE_PREFIX:-$(oc get RHMIs --all-namespaces -o json | jq -r .items[0].spec.namespacePrefix)}"
RHSSO="rhsso"
USER_SSO="user-sso"
THREESCALE="3scale"
TOKEN=$(oc whoami --show-token)
MONITORING_ROUTE=$(echo "https://$(oc get route prometheus -n ${NAMESPACE_PREFIX}observability -o=jsonpath='{.spec.host}')")/api/v1/alerts

# remove tmp files on ctrl-c
trap "rm tmp-alert-firing-during-perf-testing-report.csv tmp-alert-pending-during-perf-testing-report.csv" EXIT

# function to check if there are no alerts firing bar deadmansnitch
function CHECK_NO_ALERTS(){
    if [[ $(curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE | jq -r '.data.alerts[]| select(.state=="firing") | [.labels.alertname, .state, .activeAt ] | @csv' | wc -l  | xargs ) == 1 ]] ; then
      echo Only alert firing is DeadMansSwitch
      date
    else
      echo "============================================================================"
      echo Following alerts are Firing at :
      date
      cat alert-firing-during-perf-testing-report.csv
      echo "============================================================================"
      echo Following alerts are Pending :
      date
      cat alert-pending-during-perf-testing-report.csv
      echo "============================================================================"
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
  echo "Check $PRODUCT_NAME product namespace and operator namespace for alerts"
  while :; do
    if [ $PRODUCT_NAME == $RHSSO ] || [ $PRODUCT_NAME == $USER_SSO ]; then
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="pending") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("Keycloak.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-pending-during-perf-testing-report.csv
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="firing") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("Keycloak.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-firing-during-perf-testing-report.csv
    elif [ $PRODUCT_NAME == $THREESCALE ]; then
     curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
     | jq -r '.data.alerts[]| select((.state=="pending") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("ThreeScale.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-pending-during-perf-testing-report.csv
     curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
     | jq -r '.data.alerts[]| select((.state=="firing") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("ThreeScale.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-firing-during-perf-testing-report.csv
   else
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="pending") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'")) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-pending-during-perf-testing-report.csv
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="firing") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'")) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-firing-during-perf-testing-report.csv
    fi
    # sort command to remove duplicate alert
    sort -t, -k1 -u tmp-alert-pending-during-perf-testing-report.csv > ${PRODUCT_NAME}-alert-pending-during-perf-testing-report.csv
    sort -t, -k1 -u tmp-alert-firing-during-perf-testing-report.csv > ${PRODUCT_NAME}-alert-firing-during-perf-testing-report.csv

    CHECK_NO_ALERTS
  done
fi

