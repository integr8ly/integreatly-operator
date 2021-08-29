---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.6.0
estimate: 15m
tags:
  - automated
---

# A34 - Verify Quota values

Note: There is also an automated version of this test, so the steps below are just initial checks to validate the quota
parameter is as expected and to validate an upgrade scenario. Please also check that the A34 automated test is passing.

## Prerequisites

- Logged in to a testing cluster as a kubeadmin
- access to OCM UI org where the testing cluster is created (usually CSQE OCM org)

## Steps

1. Get `toQuota` field's value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.toQuota'
```

> The output should be `null`, unless the quota configuration is still in progress. Try again in couple of minutes.

2. Get the Quota value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
```

3. Get the param value from the Secret

```bash
oc get secret addon-managed-api-service-parameters -n redhat-rhoam-operator -o json | jq -r '.data.addon-managed-api-service' | base64 --decode
```

> Verify that the quota value matches the parameter from the secret.

> If there is no value in the secret, the cluster has been upgraded from a version of rhoam which did not have
> the quota paramater. aka pre 1.6.0. If this is the case go to step 4.

4. Get the param value from the container Environment Variable.

```bash
oc get $(oc get pods -n redhat-rhoam-operator -o name | grep rhmi-operator) -n redhat-rhoam-operator -o json | jq -r '.spec.containers[0].env[] | select(.name=="QUOTA")'
```

> Validate that the value of the status.quota matches the parameter from the secret.

5. Go to OpenShift console search for config map `quota-config-managed-api-service` in redhat-rhoam-operator namespace.
6. Depending on what the current quota value is on the testing cluster, navigate to the section of a quota config where that value is defined (e.g. for 5 million: `"name": "5"`)

7. Compare the values of resources with what you get by running this command from the terminal

```bash
backend_listener_replicas=$(oc get dc backend-listener -n redhat-rhoam-3scale --no-headers=true | awk '{print $4}')
backend_listener_resources=$(oc get dc backend-listener -n redhat-rhoam-3scale -o json | jq -r .spec.template.spec.containers[0].resources)

backend_worker_replicas=$(oc get dc backend-worker -n redhat-rhoam-3scale --no-headers=true | awk '{print $4}')
backend_worker_resources=$(oc get dc backend-worker -n redhat-rhoam-3scale -o json | jq -r .spec.template.spec.containers[0].resources)

apicast_production_replicas=$(oc get dc apicast-production -n redhat-rhoam-3scale --no-headers=true | awk '{print $4}')
apicast_production_resources=$(oc get dc apicast-production -n redhat-rhoam-3scale -o json | jq -r .spec.template.spec.containers[0].resources)

usersso_replicas=$(oc get statefulset keycloak -n redhat-rhoam-user-sso --no-headers=true | awk '{print $2}')
usersso_resources=$(oc get statefulset keycloak -n redhat-rhoam-user-sso -o json | jq -r .spec.template.spec.containers[0].resources)

ratelimit_replicas=$(oc get deployment ratelimit -n redhat-rhoam-marin3r --no-headers=true | awk '{print $2}')
ratelimit_resources=$(oc get deployment ratelimit -n redhat-rhoam-marin3r -o json | jq -r .spec.template.spec.containers[0].resources)

ratelimit_value=$(oc get configmap ratelimit-config -n redhat-rhoam-marin3r -o json | jq -r '.data["apicast-ratelimiting.yaml"]' | yq e '.descriptors[0].rate_limit.requests_per_unit'  - )

echo requests per limit value: $ratelimit_value

echo backend-listener replicas: $backend_listener_replicas
echo backend-listener resources: $backend_listener_resources

echo backend-worker replicas: $backend_worker_replicas
echo backend-worker resources: $backend_worker_resources

echo apicast-production replicas: $apicast_production_replicas
echo apicast-production resources: $apicast_production_resources

echo user-sso replicas: $usersso_replicas
echo user-sso resources: $usersso_resources

echo rate limit replicas: $ratelimit_replicas
echo rate limit resources: $ratelimit_resources
```

8. Go to OCM UI, select the testing cluster and change the quota parameter to a different value and repeat the steps from step 5. Repeat this for all available quota values.
