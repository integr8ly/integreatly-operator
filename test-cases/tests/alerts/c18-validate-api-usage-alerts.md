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

```
oc get cm rate-limit-alerts -n redhat-managed-api-operator
```

2. Verify that level1, level2 and level3 rate-limiting alerts are present

```
oc get prometheusrules -n redhat-managed-api-marin3r
```

3. Modify the `sku-limits-managed-api-service` configmap to set the rate limit to 20 requests per minute.
   Run

```
oc edit configmap sku-limits-managed-api-service -n redhat-rhmi-operator
```

And insert the following data:

```
data:
  rate_limit: |-
    {
      "RHOAM SERVICE SKU": {
        "unit": "minute",
        "requests_per_unit": 20
      }
    }
```

4. Modify the `rate-limit-alerts` to allow alerts to fire in a reasonable time from the testing perspective:

```
oc edit configmap rate-limit-alerts -n redhat-rhmi-operator
```

and insert the following data:

```
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
        "period": "2h"
      },
      "api-usage-alert-level3": {
        "ruleName": "Level3ThreeScaleApiUsageThresholdExceeded",
        "level": "warning",
        "minRate": "95%",
        "period": "30m"
      }
}
```

4. Go to `Networking` > `Routes` under `redhat-rhmi-3scale` namespace
5. Click on the `zync` route that starts with `https://3scale-admin...`
6. Go to `Secrets` > `system-seed` under 3Scale namespace and copy the admin password
7. Go back to 3Scale login page and login
8. Click on `Ok, how does 3scale work?` and follow the 3Scale wizard to create an API
9. Once on your API overview page, click on `Integration` on the left, then on `Configuration`
10. Copy the `example curl for testing` for `Staging-APIcast` and paste into a terminal window
11. Run the following script to verify that rate limit (20 requests/minute) works correctly:
```
for i in {1..21}; do <replace-with-example-curl-for-testing>; done
```
  > Only the last request should fail (TODO: any specific status code?)
12. TODO: should we update the configmap now to increase the rate limit for test the rate-limit alerts?
13. Run the following command and log in to Prometheus service (as a kubeadmin)
```
open "https://$(oc get routes prometheus-route -n redhat-rhmi-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
```
14. Go to Alerts -> and search for `Level1ThreeScaleApiUsageThresholdExceeded` alert
15. Click on the alert name -> click on the expression (link) -> Graph
16. Run the `curl` command against the APIcast endpoint so you reach between 80%-90% rate limit (i.e. amount of curl requests = rate-limit-value * 0.85)
17. Verify that the alert graph is showing some data
18. Repeat the same process for `Level2ThreeScaleApiUsageThresholdExceeded` and `Level3ThreeScaleApiUsageThresholdExceeded` alerts based on their min and max rates and verify that the alerts are firing as expected