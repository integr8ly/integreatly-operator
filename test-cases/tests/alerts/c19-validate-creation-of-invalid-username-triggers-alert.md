---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.2.0
estimate: 15m
---

# C19 - Validate creation of invalid username triggers alert

## Prerequisites

1. Login to OpenShift console and via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```shell script
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

## Steps

1. Create a new user with long name (more than 40 characters)

```bash
echo '{"apiVersion": "user.openshift.io/v1", "kind": "User", "metadata": {"name": "alongusernamethatisabovefourtycharacterslong"}}' | oc apply -f -
```

2. Go to `redhat-rhoam-rhsso` namespace
3. Search for "keycloakusers" CR
   > Validate that there is not keycloakuser that contains the name "alongusernamethatisabovefourtycharacterslong"
4. Get RHSSO admin password and note it down

```bash
oc get secret credential-rhsso -n redhat-rhoam-rhsso -o json | jq -r '.data.ADMIN_PASSWORD' | base64 --decode
```

5. Go to keycloak admin console and log in as `admin` and password from the previous step

```bash
open "https://$(oc get route keycloak -n redhat-rhoam-rhsso -o=jsonpath='{.spec.host}')"
```

6. Select `testing-idp` realm -> Users -> Add user
7. Create username `alongusernamethatisabovefourtycharacterslong2` and hit Save
8. Go to `Credentials` tab, fill in password and switch `Temporary` to OFF, hit Set password
9. In Anonymous window, go to OpenShift console and login via testing-idp as `alongusernamethatisabovefourtycharacterslong2` user
10. Go to alert manager and log in as kubeadmin

```bash
open "https://$(oc get route alertmanager-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
```

> Validate there is "ThreeScaleUserCreationFailed" alert firing
