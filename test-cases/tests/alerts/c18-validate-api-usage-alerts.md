---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.0.0
      - 1.3.0
estimate: 90m
---

# C18 - Validate API usage alerts

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```shell script
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

3. Valid SMTP credentials have been added to the `redhat-rhoam-smtp` secret

   _NOTE:_ Please reach out to Paul McCarthy <pamccart@redhat.com> for a valid SendGrid API Key

   Details on how to create a valid smtp secret can be found in this [SOP](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/install/create_cluster_smtp_configuration.md). The verification section of this SOP regarding DMS and PagerDuty configs can be skipped. Also make sure to specify the `redhat-rhoam-operator` namespace rather than `redhat-rhmi-operator`.

## Steps

1.  Validate that rate-limiting alerts ConfigMap has been created

    ```shell script
    oc get cm rate-limit-alerts -n redhat-rhoam-operator
    ```

2.  Verify that level1, level2 and level3 rate-limiting alerts are present

    ```shell script
    oc get prometheusrules -n redhat-rhoam-marin3r
    ```

3.  Modify the `sku-limits-managed-api-service` configmap to set the rate limit to 100 requests per minute.
    Run

    ```shell script
    oc patch configmap sku-limits-managed-api-service -n redhat-rhoam-operator -p '"data": {        "rate_limit": "{\n
     \"RHOAM SERVICE SKU\": {\n    \"unit\": \"minute\",\n    \"requests_per_unit\": 100,\n   \"soft_daily_limits\":
    [\n      5000000,\n      10000000,\n      15000000\n    ]\n }\n}"    }'
    ```

4.  Modify the `rate-limit-alerts` to allow alerts to fire on a per minute basis:

    ```shell script
    oc patch configmap rate-limit-alerts -n redhat-rhoam-operator -p '"data": {
    "alerts": "{\n  \"api-usage-alert-level1\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel1ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"80%\",\n      \"maxRate\": \"90%\"\n    }\n  },\n  \"api-usage-alert-level2\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel2ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"90%\",\n      \"maxRate\": \"95%\"\n    }\n  },\n  \"api-usage-alert-level3\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel3ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"95%\"\n    }\n  },\n  \"rate-limit-spike\": {\n    \"type\": \"Spike\",\n    \"level\": \"warning\",\n    \"ruleName\": \"RHOAMApiUsageOverLimit\",\n    \"period\": \"30m\"\n  }\n}"
    }'
    ```

5.  Patch the `rhoam` CR to specify BU, SRE and Customer email addresses:

    _NOTE:_ Replace `<rh_username>` references in the below commands with a valid Red Hat username. For example: `pamccart+BU@redhat.com`

    Patch BU and SRE email addresses:

    ```shell script
    oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p '{"spec":{"alertingEmailAddresses":{"businessUnit":"<rh_username>+BU@redhat.com", "cssre":"<rh_username>+SRE@redhat.com"}}}'
    ```

    Patch Customer email addresses with two emails to validate multiple addresses:

    ```shell script
    oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p '{"spec":{"alertingEmailAddress":"<rh_username>+CUSTOMER1@redhat.com <rh_username>+CUSTOMER2@redhat.com"}}'
    ```

6.  Verify email addresses are added to Alert Manager configurations

    Open the Alert Manager console and login via Openshift:

    ```shell script
    open "https://$(oc get route alertmanager-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
    ```

    Go to `Status -> Config` and check that the BU, SRE and Customer email addresses are included in the `BUAndCustomer` and `SRECustomerBU` receivers in the Alert Manager configuration where appropriate.

7.  From the Openshift console, retrieve the 3Scale admin password by going to `Secrets` > `system-seed` under the `redhat-rhoam-3scale` namespace and copying the `admin` password

8.  Next, open the 3Scale admin console and login with `admin` as the username and the retrieved password

    ```shell script
    open "https://$(oc get route -n redhat-rhoam-3scale | grep 3scale-admin | awk {'print $2'})"
    ```

9.  Click on `Ok, how does 3scale work?` and follow the 3Scale wizard to create an API

10. Once on your API overview page, click on `Integration` on the left, then on `Configuration`

11. Take note of the `example curl for testing` for `Staging-APIcast`

12. Verify RHOAMApiUsageOverLimit alert is present

    It takes 30 minutes for the RHOAMApiUsageOverLimit to fire. In a following step the number of requests required to trigger this
    alert will be reached. You will be asked to note the time at that point. Ensure the alert is present by taking the following steps:

    Navigate to the Prometheus Alert Console

    ```shell script
    open "https://$(oc get route prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')/alerts"
    ```

    Verify that the RHOAMApiUsageOverLimit alert contains `2 > 100` at the end of its query. For example:

    `max_over_time((increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m]))[30m:]) / 2 > 100`

13. Trigger API Usage alerts

        | Alert to fire                        | `numRequests` | `recipients`          |
        | ------------------------------------ | ------------- | --------------------- |
        | RHOAMApiUsageLevel1ThresholdExceeded | 43            | BU and Customers      |
        | RHOAMApiUsageLevel2ThresholdExceeded | 46            | BU and Customers      |
        | RHOAMApiUsageLevel3ThresholdExceeded | 51            | BU, Customers and SRE |

    Due to the timing of the Prometheus scrape interval, getting alerts to fire on a per 1 minute basis can be challenging. To cater for this timing issue, perform the following steps:

    Prepare the API request command below, replacing `$numrequests` with the count set in the table above. Also make sure to replace `DUMMY_URL` and `DUMMY_KEY` with the values retrieved from the 3Scale console previously

    _NOTE:_ Do not run the command yet!

    ```shell script
    for i in {1..$numRequests}; do curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>& done
    ```

    Open the Prometheus console:

    ```shell script
    open "https://$(oc get routes prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
    ```

    Add the following expression into the `Expression` field in the console:

    ```
    increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m])
    ```

    Click the `Execute` button. A `0` count should be returned.

    _NOTE:_ The next steps require preparation and need to be run quickly, please read ahead before running each step

    Next, send a request to the 3Scale, making sure to replace `DUMMY_URL` and `DUMMY_KEY` with the values retrieved from the 3Scale console earlier.

    ```shell script
    curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>
    ```

    Click the `Execute` button in the Prometheus console again until a count other than `0` is returned

    Now, every few seconds click the `Execute` button. As soon as the count goes back to `0` run the command prepared earlier in this section. This ensures that we send our requests in alignment with the Prometheus metrics scrape interval:

    ```shell script
    for i in {1..$numRequests}; do curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>& done
    ```

    Navigate back to the prometheus console, go to `Alerts` and search for the corresponding alert name in the table above e.g. `RHOAMApiUsageLevel1ThresholdExceeded`

    After a minute or so the alert should be triggered and displayed in RED

    Repeat the above steps again for each remaining alert listed in the table above

14. Once each of the above alerts have been triggered, verify that a `FIRING` and associated `RESOLVED` email notification is sent. Check the `to` field of the email to ensure that it matches the `recipients` listed in the table above. Also ensure that the link to grafana is working as expected.

15. Verify customer facing Grafana instance and Dashboard is present

    ```shell script
    open "https://$(oc get routes grafana-route -n redhat-rhoam-customer-monitoring-operator -o jsonpath='{.spec.host}')/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting"
    ```

    Navigate again to Grafana and verify that the dashboard queries and the variables are present and in the expected order.

    Run the following command:

    ```shell script
    for i in {1..1000}; do curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>& done
    ```

    _NOTE:_ The above command should eventually result in failing `429 Too Many Requests` status codes. This is to
    be expected. If no requests have been rejected make sure to check the current Rate Limit configuration in the
    `sku-limits-managed-api-service` configmap of the `redhat-rhoam-operator` namespace

    Please note the time of receiving the first 429 response in order to later verify the `RHOAMApiUsageOverLimit` Alert.

    While the command is running, check that the graphs in the Grafana dashboard are updating every minute. For example, the "Last 1 Minute - Rejected" percentage should be around 90%

    See [this example](https://user-images.githubusercontent.com/4881144/99288530-07dced00-283c-11eb-9cba-906151dd7dfb.png)

16. Verify Tier Usage Alerts

    The Tier Usage alerts are created based on the soft-limits entry in the `sku-limits-managed-api-service`

    The config map should contain `soft_daily_limits` values of 5000000, 10000000 and 15000000 after a previous patch.

    Navigate to prometheus and verify that there are 3 RHOAMApiUsageSoftLimitReachedTier Alerts present.

    ```shell script
    open "https://$(oc get route prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')/alerts"
    ```

    Verify the 3 alerts present are configured with the following values:

        | AlertName                            | `query value` | `annotations.meesage value`                      |
        | ------------------------------------ | ------------- | ------------------------------------------------ |
        | RHOAMApiUsageSoftLimitReachedTier1   | 5e+06         | soft daily limit of requests reached (5000000)   |
        | RHOAMApiUsageSoftLimitReachedTier2   | 1e+07         | soft daily limit of requests reached (10000000)  |
        | RHOAMApiUsageSoftLimitReachedTier3   | 1.5e+07       | soft daily limit of requests reached (15000000)  |

17. Verify Updating the soft_limits entry in the `sku-limits-managed-api-service` gets reflected in the Prometheus
    Alerts and Grafana Dashboard configuration

    Patch the config map with alternative soft limits to verify that the alerts and grafana dashboards get update with the new values.

    ```shell script
    oc patch configmap sku-limits-managed-api-service -n redhat-rhoam-operator -p '"data": {        "rate_limit": "{\n
     \"RHOAM SERVICE SKU\": {\n    \"unit\": \"minute\",\n    \"requests_per_unit\": 100,\n   \"soft_daily_limits\":
    [\n      15000000,\n      10000000,\n      5000000,\n      8500000\n,      18000000\n    ]\n }\n}"    }'
    ```

    Verify the alerts are present and in order according to the table below:

    Note: They will not be in order until this JIRA is completed: https://issues.redhat.com/browse/MGDAPI-681 but until it is resolved, please verify they are present.

        | AlertName                            | `query value` | `annotations.meesage value`                      |
        | ------------------------------------ | ------------- | ------------------------------------------------ |
        | RHOAMApiUsageSoftLimitReachedTier1   | 5e+06         | soft daily limit of requests reached (5000000)   |
        | RHOAMApiUsageSoftLimitReachedTier2   | 8.5e+06       | soft daily limit of requests reached (8500000)  |
        | RHOAMApiUsageSoftLimitReachedTier3   | 1e+07         | soft daily limit of requests reached (10000000)  |
        | RHOAMApiUsageSoftLimitReachedTier4   | 1.5e+07       | soft daily limit of requests reached (15000000)  |
        | RHOAMApiUsageSoftLimitReachedTier5   | 1.8e+07       | soft daily limit of requests reached (18000000)  |

    Navigate again to Grafana and verify that the dashboard queries and the variables are present and in the expected order.

    ```shell script
    open "https://$(oc get routes grafana-route -n redhat-rhoam-customer-monitoring-operator -o jsonpath='{.spec.host}')/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting"
    ```

    Log in using the credentials found in the `grafana-admin-credentials` secret in the `redhat-rhoam-customer-monitoring-operator` namespace.

    Navigate to the Rate Limiting dashboard
    Navigate to Dashboard Settings (top right - cog)
    Navigate to Variables
    They should appear as below:

    | Variable               | Definition |
    | ---------------------- | ---------- |
    | perMinuteTwentyMillion | 13889      |
    | SoftLimit1             | 3472       |
    | SoftLimit2             | 5903       |
    | SoftLimit3             | 6944       |
    | SoftLimit4             | 10417      |
    | SoftLimit5             | 12500      |

18. Verify RHOAMApiUsageOverLimit alert triggered.

    In an earlier step the presence of the RHOAMApiUsageOverLimit was verified and a time noted. If 30 minutes has passed
    since then please verify that the alert is firing and that an email has been received to the BU and Customer email.
