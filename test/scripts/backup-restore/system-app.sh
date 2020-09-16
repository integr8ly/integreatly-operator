#!/bin/sh

test_system_app () {

# Running the bundle exec script on existing system-app pod
POD_BEFORE=$(oc get pods -l deploymentConfig=system-app -o name -n redhat-rhmi-3scale | awk 'NR==1{print $1}')

echo "Executing bundle exec script in $POD_BEFORE"
kubectl exec $POD_BEFORE \
    -n redhat-rhmi-3scale \
    -- /bin/bash -c "bundle exec rake backend:storage:enqueue_rewrite"

# Scaling down system-app pods
echo "Scaling system-app pods down..."
oc scale deploymentConfigs system-app -n redhat-rhmi-3scale --replicas=0

  while true
  do
      PHASE=`oc get $POD_BEFORE -n redhat-rhmi-3scale -o template --template={{.status.phase}} 2>&1`
      if echo $PHASE | grep -q "NotFound"; then
        echo "System-app pods scaled down"
        break
      fi

      echo "Waiting for system-app pods to scale down: $PHASE..."
      sleep 5s
  done

# Scaling up system-app pods
echo "Scaling system-app pods up..."
oc scale deploymentConfigs system-app -n redhat-rhmi-3scale --replicas=2

POD_AFTER=$(oc get pods -l deploymentConfig=system-app -o name -n redhat-rhmi-3scale | awk 'NR==1{print $1}')

while true
  do
      PHASE=`oc get $POD_AFTER -n redhat-rhmi-3scale -o template --template={{.status.phase}}`
      if [ "$PHASE" = 'Running' ]; then
        break
      fi

      echo "Waiting for Redis instance to be created, current phase is: $PHASE..."
      sleep 5s
  done
}