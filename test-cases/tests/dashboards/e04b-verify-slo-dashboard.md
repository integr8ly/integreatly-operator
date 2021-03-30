---
automation:
  - INTLY-7421
components:
  - monitoring
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.2.0
      - 1.4.0
estimate: 30m
tags:
  - destructive
---

# E04B - Verify SLO dashboard

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster. Also due to known issues with the 3scale pods for these products will need to be brought back up manually.

## Steps

1. Make sure that there is at least one active alert for every panel in Grafana SLO dasboard (e.g. pod_down)
   1. Make sure rhmi-operator pod is scaled down to 0 pods
   2. Make sure all rhoam product operator pods are scaled down to 0, you can use code #1 below
   3. Make sure all keycloak stateful sets are scaled down to 0, you can use code #2 and #3 below
   4. Make sure all product pods are scaled down to 0, you can use code #4 below
2. Check the dashboard `Critical SLO summary` after some time (~20min)
   > All panels should show alerts firing
3. Bring back up the pods in the 3scale namespace
   > redhat-rhoam-3scale -> Workloads -> Deployment Configs -> Scale to 1

Execute the scripts below in a separate terminal tab/window and keep them running. It will keep scaling down resources and prevent from automatic scale up by the operator(s).

Bringing resources up might not be so simple. Make sure you have access to the terminal output of the scripts to see how many pods (replicas) were there originally and manually change that back if some alerts don't stop firing.

### Code #1

```bash
NS_PREFIX="redhat-rhoam"

namespaces=("3scale-operator" "rhsso-operator" "user-sso-operator")

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
while true; do   if oc get deployment keycloak-operator -n redhat-rhoam-rhsso-operator -o json | jq '.spec.replicas' | grep 1; then     oc scale deployment keycloak-operator --replicas=0 -n redhat-rhoam-rhsso-operator;   fi;   if oc get statefulset keycloak -n redhat-rhoam-rhsso -o json | jq '.spec.replicas' | grep 2; then     oc scale statefulset keycloak --replicas=0 -n redhat-rhoam-rhsso;   fi;   sleep 5; done
```

### Code #3

```bash
while true; do   if oc get deployment keycloak-operator -n redhat-rhoam-user-sso-operator -o json | jq '.spec.replicas' | grep 1; then     oc scale deployment keycloak-operator --replicas=0 -n redhat-rhoam-user-sso-operator;   fi;   if oc get statefulset keycloak -n redhat-rhoam-user-sso -o json | jq '.spec.replicas' | grep 3; then     oc scale statefulset keycloak --replicas=0 -n redhat-rhoam-user-sso;   fi;   sleep 5; done
```

### Code #4

```bash
NS_PREFIX="redhat-rhoam"

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
