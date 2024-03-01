---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.6.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
      - 1.20.0
      - 1.23.0
      - 1.26.0
      - 1.29.0
      - 1.32.0
      - 1.35.0
      - 1.38.0
      - 1.39.0
      - 1.40.0
estimate: 15m
---

# N10 - Verify quota feature upgrade

## Description

Measure the downtime of the RHOAM components during Quota change. Verify quota is set correctly.

## Prerequisites

- Node.js installed locally
- [oc CLI v4.3](https://docs.openshift.com/container-platform/3.6/cli_reference/get_started_cli.html#installing-the-cli)
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.CLUSTER_NAME.s1.devshift.org:6443
   ```

2. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repo and run the following command:

   ```
   git clone https://github.com/integr8ly/workload-web-app
   cd workload-web-app
   export GRAFANA_DASHBOARD=true
   make local/deploy
   ```

   > Note: do not re-deploy if the workload-web-app is already present in the cluster - check if `workload-web-app` namespace exists in the cluster or not.

   There should be no errors in the command output and product (3scale, SSO) URLS should not be blank. Alternatively, you can check the `Environment` tab in workload-webapp namespace in OpenShift console.

3. Open the RHOAM Grafana Console in the `redhat-rhoam-customer-monitoring` namespace

```bash
echo "https://$(oc get route grafana-route -n redhat-rhoam-customer-monitoring -o=jsonpath='{.spec.host}')"
```

4. Select the **Workload App** dashboard

> Verify that **3scale** and **SSO** are working by checking the **Status** graph.

5. Update the quota for a cluster in OCM to e.g. `5 million` (1 = 100k, so in this case we assign 50 to the QUOTA_VALUE variable) and wait for an operator to finish quota configuration

5.1 Login to OCM with provided token

```bash
ocm login --url=https://api.stage.openshift.com/ --token=<YOUR_TOKEN>
```

5.2 Set cluster name variable

```bash
CLUSTER_NAME="<CLUSTER_NAME>"
```

5.3 Get cluster id and assign it to a variable

```bash
CLUSTER_ID=$(ocm get clusters --parameter search="name like '%$CLUSTER_NAME%'" | jq -r '.items[].id')
```

5.4 Set quota value

```bash
QUOTA_VALUE=50 # use a different value (e.g. 10) if already on 5 million
```

5.5 Update quota

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

5.6 Check addon parameters updated successfully

```bash
ocm get /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service
```

6. After Quota change is done, open the RHOAM Grafana Console in the `redhat-rhoam-customer-monitoring` namespace

> Quota change is done when `toQuota` disappears from RHMI `rhoam` CR and `quota` is set to the expected value. Quota change takes ~1 minute.

```bash
echo "https://$(oc get route grafana-route -n redhat-rhoam-customer-monitoring -o=jsonpath='{.spec.host}')"
```

7. Select the **Workload App** dashboard

> There should be no downtime recorded during the quota change
>
> Note: it's normal that graph will show a short downtime at the start for 3scale because the workload-web-app is usually deployed before the 3scale API is ready, see [MGDAPI-1266](https://issues.redhat.com/browse/MGDAPI-1266)

8. Go to OpenShift console search for config map `quota-config-managed-api-service` in redhat-rhoam-operator namespace. See the section where a quota config for 5 million (`"name": "5"`) is defined

9. Compare the values of resources with what you get by running this command from the terminal

```bash
backend_listener_replicas=$(oc get dc backend-listener -n redhat-rhoam-3scale --no-headers=true | awk '{print $4}')
backend_listener_resources=$(oc get dc backend-listener -n redhat-rhoam-3scale -o json | jq -r '.spec.template.spec.containers[0].resources')

backend_worker_replicas=$(oc get dc backend-worker -n redhat-rhoam-3scale --no-headers=true | awk '{print $4}')
backend_worker_resources=$(oc get dc backend-worker -n redhat-rhoam-3scale -o json | jq -r '.spec.template.spec.containers[0].resources')

apicast_production_replicas=$(oc get dc apicast-production -n redhat-rhoam-3scale --no-headers=true | awk '{print $4}')
apicast_production_resources=$(oc get dc apicast-production -n redhat-rhoam-3scale -o json | jq -r '.spec.template.spec.containers[0].resources')

usersso_replicas=$(oc get statefulset keycloak -n redhat-rhoam-user-sso --no-headers=true | awk '{print $2}')
usersso_resources=$(oc get statefulset keycloak -n redhat-rhoam-user-sso -o json | jq -r '.spec.template.spec.containers[0].resources')

ratelimit_replicas=$(oc get deployment ratelimit -n redhat-rhoam-marin3r --no-headers=true | awk '{print $2}')
ratelimit_resources=$(oc get deployment ratelimit -n redhat-rhoam-marin3r -o json | jq -r '.spec.template.spec.containers[0].resources')

ratelimit_value=$(oc get configmap ratelimit-config -n redhat-rhoam-marin3r -o json | jq -r '.data["apicast-ratelimiting.yaml"]' | yq e '.[0].max_value' - )

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

echo requests per limit value: $ratelimit_value


```
