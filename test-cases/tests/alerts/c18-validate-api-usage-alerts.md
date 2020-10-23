---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 1h
tags:
  - per-release
---

# C18 - Validate API usage alerts

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. RHOAM is installed using the [Jenkins pipeline](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/ManagedAPI/job/managed-api-install-master/build?delay=0sec) with following parameters:

- integreatlyOperatorBranchName: (Go to integreatly-operator github repo and select latest RHOAM release branch)
- ocmClusterLifespan: 36
- openshiftVersion: (search for the version in the test plan document)
- useByoc: false
- multiAZ: false
- patchCloudResAwsStrCM: false
- clusterID: api-usage-alerts
- emailRecipients: <your@email.address>
- stepsToDo: provision + install

## Steps

1. Validate that rate-limiting alerts ConfigMap has been created

```sh
oc get cm rate-limit-alerts -n redhat-managed-api-operator
```

2. Verify that level1, level2 and level3 rate-limiting alerts are present

```sh
oc get prometheusrules -n redhat-managed-api-marin3r
```

3. Modify the `sku-limits-managed-api-service` configmap to set the rate limit to 20 requests per minute.
   Run

```sh
oc edit configmap sku-limits-managed-api-service -n redhat-rhmi-operator
```

And insert the following data:

```yaml
data:
  rate_limit: |-
    {
      "RHOAM SERVICE SKU": {
        "unit": "minute",
        "requests_per_unit": 100
      }
    }
```

4. Modify the `rate-limit-alerts` to allow alerts to fire in a reasonable time from the testing perspective:

```sh
oc edit configmap rate-limit-alerts -n redhat-rhmi-operator
```

and insert the following data:

```json
{
      "api-usage-alert-level1": {
        "ruleName": "Level1ThreeScaleApiUsageThresholdExceeded",
        "level": "warning",
        "minRate": "80%",
        "maxRate": "90%",
        "period": "1m"
      },
      "api-usage-alert-level2": {
        "ruleName": "Level2ThreeScaleApiUsageThresholdExceeded",
        "level": "warning",
        "minRate": "90%",
        "maxRate": "95%",
        "period": "1m"
      },
      "api-usage-alert-level3": {
        "ruleName": "Level3ThreeScaleApiUsageThresholdExceeded",
        "level": "warning",
        "minRate": "95%",
        "period": "1m"
      }
}
```

4. Go to `Networking` > `Routes` under `redhat-rhmi-3scale` namespace
5. Click on the `zync` route that starts with `https://3scale-admin...`
6. Go to `Secrets` > `system-seed` under 3Scale namespace and copy the admin password
7. Go back to 3Scale login page and login
8. Click on `Ok, how does 3scale work?` and follow the 3Scale wizard to create an API
9. Once on your API overview page, click on `Integration` on the left, then on `Configuration`
10. Take note of the `example curl for testing` for `Staging-APIcast`
11. Prepare the following script for each level to verify that the rate limit (100 requests/minute) works correctly:

    ```sh
    for i in {1..<numRequests>}; do
      curl -i <replace-with-example-curl-for-testing> &
      sleep <interval>
    done
    ```

    Replace `numRequests` and `interval` for the values from the following table:

    > These figures are obtained to make the alert fire for ~2 minutes, by calculating
    > the interval between requests to keep a rate of n request/minute where n
    > is in the range for the alert
    >
    > _Example_: To trigger level 1 we use a rate of 85 (between 80% and 90% of the limit),
    >  so we perform 170 requests every 0.71s. That is:
    >
    > * `numRequests = expected rate x minutes to keep the alert firing` 
    > * `interval = 60 / expected rate`

    | Alert to fire | `numRequests` | `interval` |
    | ------------- | ------------- | ---------- |
    | Level1ThreeScaleApiUsageThresholdExceeded | 170 | 0.71 |
    | Level2ThreeScaleApiUsageThresholdExceeded | 186 | 0.645 |
    | Level3ThreeScaleApiUsageThresholdExceeded | 240 | 0.5 |

    > Only the last requests for level 3 should fail with a `428 Too Many Requests` status code
    > due to the rate limit being exceeded
    

13. Run the following command and log in to Prometheus service (as a kubeadmin)

```sh
open "https://$(oc get routes prometheus-route -n redhat-rhmi-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
```

14. Go to Alerts -> and search for corresponding the alert (Alert to fire in the table)
15. Click on the alert name -> click on the expression (link) -> Graph
16. Run the corresponding script to trigger the alert
17. Verify that after a minute the alert graph is showing some data
    > In order to verify the current rate, run the following query in the Graph tab
    > ```
    > increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m]))
    > ```
    > The alert should fire when the value is in the expected range
18. Repeat the same process for the remaining alerts and verify that they are firing as expected. Make sure to wait at least a minute before running each script.
