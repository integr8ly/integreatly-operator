---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.0.0
      - 1.3.0
      - 1.6.0
      - 1.9.0
      - 1.11.0
      - 1.14.0
      - 1.19.0
      - 1.22.0
      - 1.25.0
      - 1.28.0
      - 1.31.0
      - 1.34.0
      - 1.37.0
      - 1.40.0
estimate: 90m
tags:
  - destructive
---

# C18 - Validate API usage alerts

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```shell script
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. k6 installed -> https://github.com/k6io/k6#install

> **NOTE:** This test case is easier read in markdown format here https://github.com/integr8ly/integreatly-operator/tree/master/test-cases/tests/alerts/c18-validate-api-usage-alerts.md

## Steps

1.  Validate that rate-limiting alerts ConfigMap has been created

    ```shell script

    oc get cm rate-limit-alerts -n redhat-rhoam-operator
    ```

2.  Verify that level1, level2 and level3 rate-limiting alerts are present

    ```shell script

    oc get prometheusrules.monitoring.rhobs -n redhat-rhoam-operator-observability | grep marin3r-api
    ```

3.  Ensure the installation is on 1 million quota to test
    Run

    ```shell script

    oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
    ```

    If the quota is not set to "1 Million" then get the cluster ID and update it using:

    ```bash

    CLUSTER_ID=$(ocm get clusters --parameter search="name like '%<your-cluster-name>%'" | jq -r '.items[].id')
    ocm patch /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service --body=<<EOF
    {
    "parameters":{
       "items":[
          {
             "id":"addon-managed-api-service",
             "value":"10"
          }
       ]
    }
    }
    EOF
    ```

    This updates the per minute rate limit to 694

4.  Modify the `rate-limit-alerts` to allow alerts to fire on a per minute basis. Note the original values to revert them back once finished:

    ```shell script

    oc patch configmap rate-limit-alerts -n redhat-rhoam-operator -p '"data": { "alerts": "{\n  \"api-usage-alert-level1\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel1ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"80%\",\n      \"maxRate\": \"90%\"\n    }\n  },\n  \"api-usage-alert-level2\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel2ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"90%\",\n      \"maxRate\": \"95%\"\n    }\n  },\n  \"api-usage-alert-level3\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel3ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"95%\"\n    }\n  },\n  \"rate-limit-spike\": {\n    \"type\": \"Spike\",\n    \"level\": \"warning\",\n    \"ruleName\": \"RHOAMApiUsageOverLimit\",\n    \"period\": \"30m\"\n  }\n}"}'
    ```

5.  Patch the `rhoam` CR to specify BU, SRE and Customer email addresses. Note the original values to revert them back once finished:

    _NOTE:_ Replace `<rh_username>` references in the below commands with a valid Red Hat username. For example: `pamccart+BU@redhat.com`.

    Patch BU and SRE email addresses:

    ```shell script

    oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p '{"spec":{"alertingEmailAddresses":{"businessUnit":"<rh_username>+BU@redhat.com", "cssre":"<rh_username>+SRE@redhat.com"}}}'
    ```

    Patch Customer email addresses with two emails to validate multiple addresses:

    ```shell script

    ocm patch /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service --body=<<EOF
    {
    "parameters":{
       "items":[
          {
             "id":"notification-email",
             "value":"<rh_username>+CUSTOMER1@redhat.com <rh_username>+CUSTOMER2@redhat.com"
          }
       ]
    }
    }
    EOF
    ```

6.  Verify email addresses are added to Alert Manager configurations

    Open the Alert Manager console and login via Openshift:

    ```shell script
     oc exec -n redhat-rhoam-operator-observability alertmanager-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9093/api/v1/status/' | jq -r '.data.configJSON.receivers[] | select(.name == "BUandCustomer") | .email_configs[].to'
    ```

    ```shell script
     oc exec -n redhat-rhoam-operator-observability alertmanager-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate 'http://localhost:9093/api/v1/status/' | jq -r '.data.configJSON.receivers[] | select(.name == "SRECustomerBU") | .email_configs[].to'
    ```

    Run those commands and verify that the BU, SRE and Customer email addresses are included in the `BUAndCustomer` and `SRECustomerBU` receivers in the Alert Manager configuration where appropriate.

7.  From the OpenShift console, retrieve the 3Scale admin password by going to `Secrets` > `system-seed` under the `redhat-rhoam-3scale` namespace and copying the `admin` password.

8.  Next, open the 3Scale admin console and login with `admin` as the username and the retrieved password. You can use Testing IDP and `customer-admin01` instead (Testing IDP password needs to be used in this case).

    ```shell script

    # For Mac
    open "https://$(oc get route -n redhat-rhoam-3scale | grep 3scale-admin | awk {'print $2'})"
    # For Linux
    xdg-open "https://$(oc get route -n redhat-rhoam-3scale | grep 3scale-admin | awk {'print $2'})"
    ```

    (If the 3scale wizard doesn't show up after accessing the 3scale webpage, update the webpage URL to "https://\<YOUR-3SCALE-ROUTE\>/p/admin/onboarding/wizard" to access the 3scale wizard)

9.  Click on `Ok, how does 3scale work?` and `Got it! Lets add my API`

10. On the page for adding a backend, you need to add a custom one. Run the following commands:

    ```bash

    oc new-project httpbin && \
    oc new-app quay.io/trepel/httpbin
    oc scale deployment/httpbin --replicas=6
    printf "\n3scale Backend Base URL: http://$(oc get svc -n httpbin --no-headers | awk '{print $3}'):8080\n"
    ```

11. Copy the `3scale Backend Base URL` to clipboard and add it to Base URL field in the 3scale wizard

12. Finish the 3scale wizard

13. Once on your API overview page, click on `Integration` on the left, then on `Configuration`. Promote to both Staging and Production

14. Take note of the `example curl for testing`. Modify it to point to Production APIcast and replace the placeholder...

    ```js
    export default function () {
      const res = http.get(
        "<3scale api stagin url from example curl request here>"
      );
    }
    ```

    ...in the integreatly-operator repo under `test-cases/tests/alerts/rate-limit.js` with the route from the curl example including the user_key param modified to point to Production APIcast

15. Verify RHOAMApiUsageOverLimit alert is present

    It takes 30 minutes for the RHOAMApiUsageOverLimit to fire. In a following step the number of requests required to trigger this
    alert will be reached. You will be asked to note the time at that point. Ensure the alert is present by taking the following steps:

    Using the command below verify that the RHOAMApiUsageOverLimit alert contains `695` at the end of its query. For example:

    ```shell script
    oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate "http://localhost:9090/api/v1/rules" | jq -r '.data.groups[].rules[] | select(.name == "RHOAMApiUsageOverLimit") | .query'
    ```

    Verify that the RHOAMApiUsageOverLimit alert contains `695` at the end of its query. For example:

    `max_over_time((increase(authorized_calls[1m]) + increase(limited_calls[1m]))[30m:]) / 695`

16. Trigger API Usage alerts

        | Alert to fire                        | `rpm` (requests per minute) | `recipients`          |
        | ------------------------------------ | --------------------------- | --------------------- |
        | RHOAMApiUsageLevel1ThresholdExceeded | 590                         | BU and Customers      |
        | RHOAMApiUsageLevel2ThresholdExceeded | 640                         | BU and Customers      |
        | RHOAMApiUsageLevel3ThresholdExceeded | 670                         | BU, Customers and SRE |

    Execute the following terminal command:

    ```shell script
    oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate "http://localhost:9090/api/v1/query?query=increase(authorized_calls%5B1m%5D)%20%2B%20increase(limited_calls%5B1m%5D)" | jq -r
    ```

    An `empty query result` should be returned.

    Navigate to the Prometheus web console (use `oc port-forward -n redhat-rhoam-operator-observability prometheus-rhoam-0 9090:9090` for that), go to `Alerts` and search for the corresponding alert name in the table above e.g. `RHOAMApiUsageLevel1ThresholdExceeded`

    Run a separate load test for each rpm from the RHOAMApiUsageLevel<level No.>ThresholdExceeded using k6

    ```shell script

    # RHOAMApiUsageLevel1ThresholdExceeded
    export rpm=590
    RPM=$rpm k6 run test-cases/tests/alerts/rate-limit.js
    # RHOAMApiUsageLevel2ThresholdExceeded
    export rpm=640
    RPM=$rpm k6 run test-cases/tests/alerts/rate-limit.js
    # RHOAMApiUsageLevel3ThresholdExceeded
    export rpm=670
    RPM=$rpm k6 run test-cases/tests/alerts/rate-limit.js
    ```

    After a minute or two the alert should be triggered and displayed in RED, You will need to refresh Prometheus in the browser for it to update.

    Interrupt the script (Ctrl+C), wait 1 minute and repeat for the next rpm

17. Once each of the above alerts have been triggered, verify that a `FIRING` and associated `RESOLVED` email notification is sent (these might not be sent). Check the `to` field of the email to ensure that it matches the `recipients` listed in the table above. Also ensure that the link to grafana is working as expected.

18. Verify customer facing Grafana instance and Dashboard is present

    ```shell script

    # For Mac
    open "https://$(oc get routes grafana-route -n redhat-rhoam-customer-monitoring -o jsonpath='{.spec.host}')/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting"
    # For Linux
    xdg-open "https://$(oc get routes grafana-route -n redhat-rhoam-customer-monitoring -o jsonpath='{.spec.host}')/d/66ab72e0d012aacf34f907be9d81cd9e/rate-limiting"
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
    since then please verify that the alert is firing and that an email has been received to the BU and Customer email (both for firing and for resolving).

20. Revert the changes back

    - revert back the quota
    - revert back the email addresses
    - revert back the alert period length
