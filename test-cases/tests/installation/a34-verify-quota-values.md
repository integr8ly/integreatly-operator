---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.6.0
      - 1.10.0
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

> The output should be `null`, unless the quota configuration is still in progress. Try again in a couple of minutes.

2. Get the Quota value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
```

3. Get the param value from the Secret

```bash
oc get secret addon-managed-api-service-parameters -n redhat-rhoam-operator -o json | jq -r '.data.addon-managed-api-service' | base64 --decode
```

> Verify that the quota value matches the mapped parameter from the secret.
>
> if param is == 0 then Quota should be 100K - Evaluation -> (Evaluation option)
> if param is == 10 then Quota should be 1 Million
> if param is == 50 then Quota should be 5 Million
> if param is == 100 then Quota should be 10 Million
> if param is == 200 then Quota should be 20 Million
> if param is == 500 then Quota should be 50 Million
> if param is == 1 then Quota should be 100k

> If there is no value in the secret, the cluster has been upgraded from a version of rhoam which did not have
> the quota paramater. aka pre 1.6.0. If this is the case go to step 4. Otherwise skip to step 5.

4. Get the param value from the container Environment Variable.

```bash
oc get $(oc get pods -n redhat-rhoam-operator -o name | grep rhmi-operator) -n redhat-rhoam-operator -o json | jq -r '.spec.containers[0].env[] | select(.name=="QUOTA")'
```

> Validate that the value of the status.quota matches the parameter from the secret using the mapping above.

5. Go to OpenShift console search for config map `quota-config-managed-api-service` in redhat-rhoam-operator namespace.
6. Depending on what the current quota value is on the testing cluster, navigate to the section of a quota config where that value is defined (e.g. for 5 million: `"param": "50"`)

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

ratelimit_value=$(oc get configmap ratelimit-config -n redhat-rhoam-marin3r -o json | jq -r '.data["apicast-ratelimiting.yaml"]' | yq e '.[0].max_value' - )

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

8. Login to OCM with provided token

```bash
ocm login --url=https://api.stage.openshift.com/ --token=<YOUR_TOKEN>
```

9. Set cluster name variable

```bash
CLUSTER_NAME="<CLUSTER_NAME>"
```

10. Get cluster id and assign it to a variable

```bash
CLUSTER_ID=$(ocm get clusters --parameter search="name like '%$CLUSTER_NAME%'" | jq -r '.items[].id')
```

11. Set quota value

```bash
QUOTA_VALUE=<QUOTA_VALUE>
```

12. Update quota

```bash
ocm patch /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service --body=<<EOF
{
   "parameters":{
      "items":[
         {
            "id":"addon-managed-api-service",
            "value":"$QUOTA_VALUE"
         }
      ]
   }
}
EOF
```

13. Check addon parameters updated successfully

```bash
ocm get /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service
```

14. Repeat this for all available quota values.
