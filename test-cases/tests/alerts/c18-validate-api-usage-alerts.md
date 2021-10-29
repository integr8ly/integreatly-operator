---
products:
  - name: rhoam
    environments:
      - external
    targets:
      - 1.0.0
      - 1.3.0
      - 1.6.0
      - 1.9.0
      - 1.11.0
      - 1.14.0
estimate: 90m
---

# C18 - Validate API usage alerts

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```shell script
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. RHOAM is installed using the [Jenkins pipeline](https://master-jenkins-csb-intly.apps.ocp4.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-master/build?delay=0sec) with following parameters:

   - integreatlyOperatorBranchName: (Go to integreatly-operator github repo and select latest RHOAM release branch)
   - ocmClusterLifespan: 36
   - openshiftVersion: (search for the version in the test plan document)
   - useByoc: false
   - quota: 10
   - multiAZ: false
   - patchCloudResAwsStrCM: false
   - clusterID: api-usage-alerts
   - emailRecipients: <your@email.address>
   - pipelineSteps: provisionCluster, installProduct, setupIdp

3. Valid SMTP credentials have been added to the `redhat-rhoam-smtp` secret

   _NOTE:_ Please reach out to Paul McCarthy <pamccart@redhat.com> for a valid SendGrid API Key

   Details on how to create a valid smtp secret can be found in this [SOP](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/install/create_cluster_smtp_configuration.md). The verification section of this SOP regarding DMS and PagerDuty configs can be skipped. Also make sure to specify the `redhat-rhoam-operator` namespace rather than `redhat-rhmi-operator`.

4. k6 installed -> https://github.com/k6io/k6#install

## Steps

1.  Validate that rate-limiting alerts ConfigMap has been created

    ```shell script
    oc get cm rate-limit-alerts -n redhat-rhoam-operator
    ```

2.  Verify that level1, level2 and level3 rate-limiting alerts are present

    ```shell script
    oc get prometheusrules -n redhat-rhoam-marin3r
    ```

3.  Ensure the installation is on 1 million quota to test
    Run

    ```shell script
    oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
    ```

    If the quota is not set to "1", then update it using:

    ```shell script
    INSTALLATION_TYPE=managed-api make cluster/prepare/quota DEV_QUOTA="10"
    ```

    This updates the per minute rate limit to 694

4.  Modify the `rate-limit-alerts` to allow alerts to fire on a per minute basis:

    ```shell script
    oc patch configmap rate-limit-alerts -n redhat-rhoam-operator -p '"data": { "alerts": "{\n  \"api-usage-alert-level1\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel1ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"80%\",\n      \"maxRate\": \"90%\"\n    }\n  },\n  \"api-usage-alert-level2\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel2ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"90%\",\n      \"maxRate\": \"95%\"\n    }\n  },\n  \"api-usage-alert-level3\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel3ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"95%\"\n    }\n  },\n  \"rate-limit-spike\": {\n    \"type\": \"Spike\",\n    \"level\": \"warning\",\n    \"ruleName\": \"RHOAMApiUsageOverLimit\",\n    \"period\": \"30m\"\n  }\n}"}'
    ```

5.  Patch the `rhoam` CR to specify BU, SRE and Customer email addresses:

    _NOTE:_ Replace `<rh_username>` references in the below commands with a valid Red Hat username. For example: `pamccart+BU@redhat.com`.

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
    open "https://$(oc get route alertmanager-route -n redhat-rhoam-observability -o jsonpath='{.spec.host}')"
    ```

    Go to `Status -> Config` and check that the BU, SRE and Customer email addresses are included in the `BUAndCustomer` and `SRECustomerBU` receivers in the Alert Manager configuration where appropriate.

7.  From the OpenShift console, retrieve the 3Scale admin password by going to `Secrets` > `system-seed` under the `redhat-rhoam-3scale` namespace and copying the `admin` password

8.  Next, open the 3Scale admin console and login with `admin` as the username and the retrieved password

    ```shell script
    open "https://$(oc get route -n redhat-rhoam-3scale | grep 3scale-admin | awk {'print $2'})"
    ```

    (If the 3scale wizard doesn't show up after accessing the 3scale webpage, update the webpage URL to "https://\<YOUR-3SCALE-ROUTE\>/p/admin/onboarding/wizard" to access the 3scale wizard)

9.  Click on `Ok, how does 3scale work?` and `Got it! Lets add my API`

10. On the page for adding a backend, you need to add a custom one. Run the following commands:

```bash
oc new-project httpbin && \
oc new-app jsmadis/httpbin && \
oc scale deployment/httpbin --replicas=6 && \
printf "\n3scale Backend Base URL: http://$(oc get svc -n httpbin --no-headers | awk '{print $3}'):8080\n"
```

11. Copy the `3scale Backend Base URL` to clipboard and add it to Base URL field in the 3scale wizard

12. Finish the 3scale wizard

13. Once on your API overview page, click on `Integration` on the left, then on `Configuration`

14. Take note of the `example curl for testing` and replace the placeholder in `test-cases/tests/alerts/rate-limit.js` with the route from the curl example including the user_key param

15. Verify RHOAMApiUsageOverLimit alert is present

    It takes 30 minutes for the RHOAMApiUsageOverLimit to fire. In a following step the number of requests required to trigger this
    alert will be reached. You will be asked to note the time at that point. Ensure the alert is present by taking the following steps:

    Navigate to the Prometheus Alert Console

    ```shell script
    open "https://$(oc get route prometheus-route -n redhat-rhoam-observability -o jsonpath='{.spec.host}')/alerts"
    ```

    Verify that the RHOAMApiUsageOverLimit alert contains `694` at the end of its query. For example:

    `max_over_time((increase(authorized_calls[1m]) + increase(limited_calls[1m]))[30m:]) / 694`

16. Trigger API Usage alerts

        | Alert to fire                        | `rpm` (requests per minute) | `recipients`          |
        | ------------------------------------ | --------------------------- | --------------------- |
        | RHOAMApiUsageLevel1ThresholdExceeded | 590                         | BU and Customers      |
        | RHOAMApiUsageLevel2ThresholdExceeded | 640                         | BU and Customers      |
        | RHOAMApiUsageLevel3ThresholdExceeded | 670                         | BU, Customers and SRE |

    Open the Prometheus console:

    ```shell script
    open "https://$(oc get routes prometheus-route -n redhat-rhoam-observability -o jsonpath='{.spec.host}')"
    ```

    Add the following expression into the `Expression` field in the console:

    ```
    increase(authorized_calls[1m]) + increase(limited_calls[1m])
    ```

    Click the `Execute` button. A `0` count should be returned.

    Navigate back to the prometheus console, go to `Alerts` and search for the corresponding alert name in the table above e.g. `RHOAMApiUsageLevel1ThresholdExceeded`

    Run the command `RPM=$rpm k6 run test-cases/tests/alerts/rate-limit.js`

    After a minute or so the alert should be triggered and displayed in RED

    Interrupt the script (Ctrl+C), wait 1 minute and repeat the above steps again for each remaining alert listed in the table above

17. Once each of the above alerts have been triggered, verify that a `FIRING` and associated `RESOLVED` email notification is sent. Check the `to` field of the email to ensure that it matches the `recipients` listed in the table above. Also ensure that the link to grafana is working as expected.

18. Verify customer facing Grafana instance and Dashboard is present

    ```shell script
    open "https://$(oc get routes grafana-route -n redhat-rhoam-customer-monitoring-operator -o jsonpath='{.spec.host}')/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting"
    ```

    Navigate again to Grafana and verify that the dashboard queries and the variables are present and in the expected order.

    Run the following command:

    ```shell script
    RPM=750 k6 run test-cases/tests/alerts/rate-limit.js
    ```

    _NOTE:_ The above command should eventually result in failing `429 Too Many Requests` status codes. This is to
    be expected. If no requests have been rejected make sure to check the current Rate Limit configuration in the
    `ratelimit-config` configmap of the `redhat-rhoam-marin3r` namespace

    Please note the time of receiving the first 429 response in order to later verify the `RHOAMApiUsageOverLimit` Alert.

    While the command is running, check that the graphs in the Grafana dashboard are updating every minute. For example, the "Last 1 Minute - Rejected" percentage should be around 10%

    See [this example](https://user-images.githubusercontent.com/4881144/99288530-07dced00-283c-11eb-9cba-906151dd7dfb.png)

19. Verify RHOAMApiUsageOverLimit alert triggered.

    In an earlier step the presence of the RHOAMApiUsageOverLimit was verified and a time noted. If 30 minutes has passed
    since then please verify that the alert is firing and that an email has been received to the BU and Customer email.
