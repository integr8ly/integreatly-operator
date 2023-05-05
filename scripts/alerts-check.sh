#!/bin/bash
# USAGE
# ./alerts-check.sh <optional sleep in seconds> <optional product-name>
# ^C to break
# Generates two files one for firing and one for pending alerts
#
# PREREQUISITES
# - jq
# - oc (logged in at the cmd line in order to get the bearer token)
# VARIABLES
NAMESPACE_PREFIX="${NAMESPACE_PREFIX:-$(oc get RHMIs --all-namespaces -o jsonpath='{.items[0].spec.namespacePrefix}')}"
RHSSO="rhsso"
USER_SSO="user-sso"
THREESCALE="3scale"
MONITORING_ROUTE=$(echo "https://$(oc get route prometheus -n ${NAMESPACE_PREFIX}observability -o=jsonpath='{.spec.host}')")/api/v1/alerts
REPORT_PREFIX=$(date +"%Y-%m-%d-%H-%M")

SLEEP_TIME="${1-5}"
TOKEN=$(oc whoami --show-token)
PRODUCT_NAME=`echo $2 | grep "3scale\|user-sso\|rhsso\|marin3r"`

# remove tmp files on ctrl-c
trap "rm tmp-alert-firing-report.csv tmp-alert-pending-report.csv" EXIT

# function to check if there are no alerts firing bar deadmansnitch
function CHECK_NO_ALERTS(){
    if [[ $(curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE | jq -r '.data.alerts[]| select(.state=="firing") | [.labels.alertname, .state, .activeAt, .labels.severity ] | @csv' | wc -l  | xargs ) == 1 ]] ; then
      echo Only alert firing is DeadMansSwitch
      date
    else
      echo "============================================================================"
      echo Following alerts are Firing at :
      date
      cat ${REPORT_PREFIX}${PRODUCT_NAME}-alert-firing-report.csv
      echo "============================================================================"
      echo Following alerts are Pending :
      date
      cat ${REPORT_PREFIX}${PRODUCT_NAME}-alert-pending-report.csv
      echo "============================================================================"
    fi

    sleep $SLEEP_TIME
    if [[ $? != 0 ]]; then
        sleep 5
    fi
}

# If product name not passed in then run for all products
if [[ -z "$PRODUCT_NAME" ]]; then
  while :; do
    # generate a report
    curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
    | jq -r '.data.alerts[]| select(.state=="pending") | [.labels.alertname, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-pending-report.csv

    curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
    | jq -r '.data.alerts[]| select(.state=="firing") | [.labels.alertname, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-firing-report.csv

    # sort command to remove duplicate alert
    sort -t',' -k 1,1 -u tmp-alert-firing-report.csv > ${REPORT_PREFIX}-alert-firing-report.csv
    #sort -t, -k1 -u alert-firing-report.csv > alert-firing-report.csv
    sort -t, -k1 -u tmp-alert-pending-report.csv > ${REPORT_PREFIX}-alert-pending-report.csv
    CHECK_NO_ALERTS
  done
fi

# Check the alerts in product namespaces
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
      | jq -r '.data.alerts[]| select((.state=="pending") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("Keycloak.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-pending-report.csv
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="firing") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("Keycloak.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-firing-report.csv
    elif [ $PRODUCT_NAME == $THREESCALE ]; then
     curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
     | jq -r '.data.alerts[]| select((.state=="pending") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("ThreeScale.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-pending-report.csv
     curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
     | jq -r '.data.alerts[]| select((.state=="firing") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'" or (.labels.alertname|test("ThreeScale.+")))) | [.labels.alertname, .labels.namespace, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-firing-report.csv
   else
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="pending") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'")) | [.labels.alertname, .labels.namespace, .state, .activeAt ] | @csv'>> tmp-alert-pending-report.csv
      curl -s -H "Authorization: Bearer $TOKEN" $MONITORING_ROUTE \
      | jq -r '.data.alerts[]| select((.state=="firing") and (.labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'" or .labels.namespace=="'$NAMESPACE_PREFIX''$PRODUCT_NAME'-operator" or .labels.productName=="'$PRODUCT_NAME'")) | [.labels.alertname, .labels.namespace, .state, .activeAt, .labels.severity ] | @csv'>> tmp-alert-firing-report.csv
    fi
    # sort command to remove duplicate alert
    sort -t, -k1 -u tmp-alert-pending-report.csv > ${REPORT_PREFIX}${PRODUCT_NAME}-alert-pending-report.csv
    sort -t, -k1 -u tmp-alert-firing-report.csv > ${REPORT_PREFIX}${PRODUCT_NAME}-alert-firing-report.csv

    CHECK_NO_ALERTS
  done
fi
