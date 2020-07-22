---
estimate: 30m
automation_jiras:
  - INTLY-7421
---

# E04 - Verify SLO dashboard

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster. Also due to known issues with the 3scale, CodeReady, Solution Explorer, and UPS operators pods for these products will need to be brought back up manually.

## Steps

1. Make sure that there is at least one active alert for every panel in SLO dasboard (e.g. pod_down)
   1. Make sure rhmi-operator pod is scaled down to 0 pods
   2. Make sure all rhmi product operator pods are scaled down to 0, you can use code #1 below
   3. Make sure all keycloak stateful sets are scaled down to 0, you can use code #2 and #3 below
   4. Make sure all product pods are scaled down to 0, you can use code #4 below
2. Check the dashboard `Critical SLO summary` after some time (~20min)
   > All panels should show alerts firing
3. Bring back up the pods in the 3scale, CodeReady, Solution Explorer, and UPS namespaces
   > redhat-rhmi-3scale -> Workloads -> Deployment Configs -> Scale to 1
   > redhat-rhmi-codeready-workspaces -> Workloads -> Deployments -> Scale to 1
   > redhat-rhmi-solution-explorer -> Workloads -> Deployment Configs -> Scale to 1
   > redhat-rhmi-ups -> Workloads -> Deployments -> Scale to 1

**_Note_**: The alerts firing panel in the SLO summary dashboard may show alerts firing even if the product is no longer triggering an alert in prometheus.
If the alert box shows alerts firing, change the `quick range` in the top right of Grafana UI to a lower range (such as 5 minutes) and check the `alert firing` box again and ensure
that the graph is true to the actual state of the alerts in prometheus, if the `alert firing` box is no longer showing alerts and the graph is true to the actual state then no further action is required.

### Code #1

```bash
NS_PREFIX="redhat-rhmi"

namespaces=("3scale-operator" "amq-online" "apicurito-operator" "codeready-workspaces-operator" "fuse-operator" "rhsso-operator" "solution-explorer-operator" "ups-operator" "user-sso-operator")

while true; do
  echo "scaling down"

  for proj in ${namespaces[@]}; do

    project="$NS_PREFIX-$proj"

    for deployment in `oc get deployment -n $project -o json |  jq -r '.items[].metadata.name' | grep operator`; do
      if oc get deployment $deployment -n $project -o json | jq '.spec.replicas' | grep -v 0; then
        echo "scaling down $deployment"
        oc scale deployment $deployment --replicas=0 -n $project
      fi
    done

    for dc in `oc get dc -n $project -o json |  jq -r '.items[].metadata.name' | grep operator`; do
      if oc get dc $dc -n $project -o json | jq '.spec.replicas' | grep -v 0; then
        echo "scaling down $dc"
        oc scale dc $dc --replicas=0 -n $project
      fi
    done

  done
done
```

### Code #2

```bash
while true; do   if oc get deployment keycloak-operator -n redhat-rhmi-rhsso-operator -o json | jq '.spec.replicas' | grep 1; then     oc scale deployment keycloak-operator --replicas=0 -n redhat-rhmi-rhsso-operator;   fi;   if oc get statefulset keycloak -n redhat-rhmi-rhsso -o json | jq '.spec.replicas' | grep 2; then     oc scale statefulset keycloak --replicas=0 -n redhat-rhmi-rhsso;   fi;   sleep 5; done
```

### Code #3

```bash
while true; do   if oc get deployment keycloak-operator -n redhat-rhmi-user-sso-operator -o json | jq '.spec.replicas' | grep 1; then     oc scale deployment keycloak-operator --replicas=0 -n redhat-rhmi-user-sso-operator;   fi;   if oc get statefulset keycloak -n redhat-rhmi-user-sso -o json | jq '.spec.replicas' | grep 2; then     oc scale statefulset keycloak --replicas=0 -n redhat-rhmi-user-sso;   fi;   sleep 5; done
```

### Code #4

```bash
NS_PREFIX="redhat-rhmi"

namespaces=("3scale" "amq-online" "apicurito" "codeready-workspaces" "fuse" "rhsso" "solution-explorer" "ups" "user-sso")

while true; do
  echo "scaling down"

  for proj in ${namespaces[@]}; do

    project="$NS_PREFIX-$proj"

    for deployment in `oc get deployment -n $project -o json |  jq -r '.items[].metadata.name' | grep -v operator`; do
      if oc get deployment $deployment -n $project -o json | jq '.spec.replicas' | grep -v 0; then
        echo "scaling down $deployment"
        oc scale deployment $deployment --replicas=0 -n $project
      fi
    done

    for dc in `oc get dc -n $project -o json |  jq -r '.items[].metadata.name' | grep -v operator`; do
      if oc get dc $dc -n $project -o json | jq '.spec.replicas' | grep -v 0; then
        echo "scaling down $dc"
        oc scale dc $dc --replicas=0 -n $project
      fi
    done

  done
done
```
