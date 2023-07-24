#!/bin/bash

# Run this script before triggering an upgrade to check how long the monitoring upgrade takes.
#

#Log time when upgrade begins
echo "checking for beginning of upgrade"
while true
do
  toVersion=$(oc get rhmi rhoam -n redhat-rhoam-operator -o yaml | yq e '.status.toVersion' -)
  if [ "$toVersion" == "1.13.0" ]; then
    date +"%T"
    echo "upgrade has begun"
    break
  fi
  sleep 30
done

# check both stateful sets for prometheus and alertmananger in the AMO namespace.
# when both become unavailable this indicates the unavailability of the uninstalling monitoring stack
# The time is noted and stored for use later
while true
do
  oc get statefulsets prometheus-application-monitoring -n redhat-rhoam-middleware-monitoring-operator
  if [ "$?" == "1" ]; then
    echo "Prometheus is removed - moving on as either prometheus or alertmanager down means the stack has begun to uninstall"
    date +"%T"
    START=$(date +%s);
    break
  fi
  oc get statefulsets alertmanager-application-monitoring -n redhat-rhoam-middleware-monitoring-operator
  if [ "$?" == "1" ]; then
    date +"%T"
    echo "AlertManager is removed - moving on as either prometheus or alertmanager down means the stack has begun to uninstall"
    START=$(date +%s);
    break
  fi
  sleep 15
done

# check both statefulsets (readyReplicas) for prometheus and alertmananger in the Observability namespace.
# when both become available this indicates the availability of the installing monitoring stack
# The time is noted and stored for use later
while true
do
  ooPrometheus=$(oc get statefulset prometheus-rhoam -n redhat-rhoam-operator-observability -o yaml | yq e '.status.readyReplicas' -)
  if [ "$ooPrometheus" -ge 1 ]; then
    echo "Prometheus reporting 1 replica ready"
    ooAlertManager=$(oc get statefulset alertmanager-rhoam -n redhat-rhoam-operator-observability -o yaml | yq e '.status.readyReplicas' -)
    if [ "$ooAlertManager" -ge 1 ]; then
      echo "AlertManager reporting 2 replica ready"
      date +"%T"
      END=$(date +%s);
      break
    fi
  fi
  sleep 15
done

echo "The time the upgrade of monitoring took: "
echo $((END-START)) | awk '{print int($1/60)":"int($1%60)}'
