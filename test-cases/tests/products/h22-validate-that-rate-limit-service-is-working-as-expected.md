---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - osd-private-post-upgrade
estimate: 1h
tags:
  - per-release
---

# H22 - Validate that Rate Limit service is working as expected

## Description

This test case should prove that the rate limiting Redis counter correctly increases with every request made

## Steps

1. Open Openshift Console in your browser
2. Copy `oc login` command and login to your cluster in the terminal
3. Go to `Networking` > `Routes` under `redhat-rhoam-3scale` namespace
4. Click on the `zync` route that starts with `https://3scale-admin...`
5. Go to `Secrets` > `system-seed` under 3Scale namespace and copy the admin password
6. Go back to 3Scale login page and login
7. Click on `Ok, how does 3scale work?` and follow the 3Scale wizard to create an API
8. Once on your API overview page, click on `Integration` on the left, then on `Configuration`
9. Copy the `example curl for testing` for `Staging-APIcast` and paste into a terminal window
10. Create and run the following script:

```
#!/bin/sh
echo "Creating throwaway redis container..."
  cat << EOF | oc create -f - -n redhat-rhoam-operator
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

  while true
  do
      PHASE=`oc get redis/throw-away-redis-pod -n redhat-rhoam-operator -o template --template={{.status.phase}}`
      if [ "$PHASE" = 'complete' ]; then
        break
      fi

      echo "Waiting for throwaway Postgres container to complete. Current phase: $PHASE..."
      sleep 10s
  done

echo "Container ready..."
echo ""
echo "Running Redis connections count for 30 minutes..."
echo ""

POD_NAME=$(oc get pod -n redhat-rhoam-operator | grep -v "deploy" | grep throw-away-redis-pod | awk '{print $1}')
REDIS_HOST=$(oc get secrets/ratelimit-service-redis-managed-api -n redhat-rhoam-operator -o template --template={{.data.uri}} | base64 --decode)

OS=$(uname)
if [[ $OS = Linux ]]; then
    RUNTIME="30 minute"
    ENDTIME=$(date -ud "$RUNTIME" +%s)
elif [[ $OS = Darwin ]]; then
    RUNTIME="+30m"
    ENDTIME=$(date -v $RUNTIME +%s)
fi

while [[ $(date -u +%s) -le $ENDTIME ]]
do
RESULT=$(kubectl exec $POD_NAME \
    -n redhat-rhoam-operator \
    -- /opt/rh/rh-redis32/root/usr/bin/redis-cli -c -h $REDIS_HOST -p 6379 KEYS '*' | grep -v "liveness-probe")

    if (( $(grep -c . <<<"$RESULT") > 1 )); then
        echo "Flushing REDIS, time slot for allowed hits per unit has been reached. Ping APICAST again to create a new key value pair"
        kubectl exec $POD_NAME \
            -n redhat-rhoam-operator \
            -- /opt/rh/rh-redis32/root/usr/bin/redis-cli -c -h $REDIS_HOST -p 6379 FLUSHALL
    else
        HITCOUNT=$(kubectl exec $POD_NAME \
            -n redhat-rhoam-operator \
            -- /opt/rh/rh-redis32/root/usr/bin/redis-cli -c -h $REDIS_HOST -p 6379 GET $RESULT)

        if echo $HITCOUNT | grep -q "ERR";
        then
            echo ""
            echo "Curl Apicast endpoint to create a REDIS key value pair"
            echo ""
        else
            echo ""
            echo "REDIS key in rate limit redis: $RESULT"
            echo "Current hit count: $HITCOUNT"
            echo ""
        fi
fi
    echo "-------------------------------------------------------------------------------"
    sleep 5s
done
```

11. The script will create a throw away REDIS container and it will communicate with REDIS rate limiting pod. It will run for
    30 minutes and every 5 seconds it will output a key value pair stored in Redis.
12. With script running, CURL the apicast endpoint, you should see that the amount of hits increases with the amount of CURL commands
    executed by you. Every 1 minute, REDIS will be flushed in order to start a new counting cycle.
13. Once verified that the REDIS hitcount increases, shut down the script and run the following command to remove redis throw-away pod

```
oc delete redis/throw-away-redis-pod -n redhat-rhoam-operator
```
