---
automation:
  - MGDAPI-1260
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.2.0
      - 1.6.0
      - 1.9.0
tags:
  - automated
  - destructive
estimate: 15m
---

# C19 - Validate creation of invalid username triggers alert

## Prerequisites

1. Login to OpenShift console and via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```shell script
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. Access to OCM QE console

## Steps

### Automated Test

- To run the automated test manually that performs the manual steps below:

```
LOCAL=false INSTALLATION_TYPE=managed-api BYPASS_STORAGE_TYPE_CHECK=true TEST="C19" make test/e2e/single | tee c19-test.log
```

- Verify test completes successfully

### Manual Steps

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
oc exec -n redhat-rhoam-operator-observability alertmanager-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate http://localhost:9093/api/v1/alerts | jq -r
```

> Validate there is "ThreeScaleUserCreationFailed" alert firing

11. Delete the user from OpenShift

```bash
oc delete user alongusernamethatisabovefourtycharacterslong2
```

12. Go back to the alert manager
    > Validate that the alert is no longer firing (it might take some time)
13. In a new anonymous window, login to the cluster via testing-idp as customer-admin01 user
14. Get 3scale admin password and note it down

```bash
oc get secret system-seed -n redhat-rhoam-3scale -o json | jq -r '.data.ADMIN_PASSWORD' | base64 --decode
```

15. In OpenShift console, go to 3scale namespace -> Routes and navigate to 3scale admin console
16. Log in as `admin` and use the password from the previous step
17. Go to settings -> Users -> Listing
    > Verify that customer-admin01 user is listed there
18. Go to OCM UI console and search for the testing cluster
19. Go to Access control and delete testing-idp and all customer admin users
20. Go back to 3scale admin console and refresh the page with user listing
    > Verify that the customer-admin01 user is no longer present in 3scale
