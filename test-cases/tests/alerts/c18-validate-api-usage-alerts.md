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

3. Valid SMTP credentials have been added to the `redhat-rhoam-smtp` secret

   _NOTE:_ Please reach out to Paul McCarthy <pamccart@redhat.com> for a valid SendGrid API Key

   Details on how to create a valid smtp secret can be found in this [SOP](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/install/create_cluster_smtp_configuration.md). The verification section of this SOP regarding DMS and PagerDuty configs can be skipped. Also make sure to specify the `redhat-rhoam-operator` namespace rather than `redhat-rhmi-operator`.

## Steps

1. Validate that rate-limiting alerts ConfigMap has been created

   ```sh
   oc get cm rate-limit-alerts -n redhat-rhoam-operator
   ```

2. Verify that level1, level2 and level3 rate-limiting alerts are present

   ```sh
   oc get prometheusrules -n redhat-rhoam-marin3r
   ```

3. Modify the `sku-limits-managed-api-service` configmap to set the rate limit to 100 requests per minute.
   Run

   ```sh
   oc patch configmap sku-limits-managed-api-service -n redhat-rhoam-operator -p '"data": {        "rate_limit": "{\n  \"RHOAM SERVICE SKU\": {\n    \"unit\": \"minute\",\n    \"requests_per_unit\": 100\n  }\n}"    }'
   ```

4. Modify the `rate-limit-alerts` to allow alerts to fire on a per minute basis:

   ```sh
   oc patch configmap rate-limit-alerts -n redhat-rhoam-operator -p '"data": {
       "alerts": "{\n  \"api-usage-alert-level1\": {\n    \"ruleName\": \"RHOAMApiUsageLevel1ThresholdExceeded\",\n    \"level\": \"warning\",\n    \"minRate\": \"80%\",\n    \"maxRate\": \"90%\",\n    \"period\": \"1m\"\n  },\n  \"api-usage-alert-level2\": {\n    \"ruleName\": \"RHOAMApiUsageLevel2ThresholdExceeded\",\n    \"level\": \"warning\",\n    \"minRate\": \"90%\",\n    \"maxRate\": \"95%\",\n    \"period\": \"1m\"\n  },\n  \"api-usage-alert-level3\": {\n    \"ruleName\": \"RHOAMApiUsageLevel3ThresholdExceeded\",\n    \"level\": \"warning\",\n    \"minRate\": \"95%\",\n    \"period\": \"1m\"\n  }\n}"
   }'
   ```

5. Patch the `rhoam` CR to specify BU, SRE and Customer email addresses:

   _NOTE:_ Replace `<rh_username>` references in the below commands with a valid Red Hat username. For example: `pamccart+BU@redhat.com`

   Patch BU and SRE email addresses:

   ```sh
   oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p '{"spec":{"alertingEmailAddresses":{"businessUnit":"<rh_username>+BU@redhat.com", "cssre":"<rh_username>+SRE@redhat.com"}}}'
   ```

   Patch Customer email addresses with two emails to validate multiple addresses:

   ```sh
   oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p '{"spec":{"alertingEmailAddress":"<rh_username>+CUSTOMER1@redhat.com <rh_username>+CUSTOMER2@redhat.com"}}'
   ```

6. Verify email addresses are added to Alert Manager configurations

   Open the Alert Manager console and login via Openshift:

   ```sh
   open "https://$(oc get route alertmanager-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
   ```

   Go to `Status -> Config` and check that the BU, SRE and Customer email addresses are included in the `BUAndCustomer` and `SRECustomerBU` receivers in the Alert Manager configuration where appropriate.

7. From the Openshift console, retrieve the 3Scale admin password by going to `Secrets` > `system-seed` under the `redhat-rhoam-3scale` namespace and copying the `admin` password

8. Next, open the 3Scale admin console and login with `admin` as the username and the retrieved password

   ```sh
   open "https://$(oc get route -n redhat-rhoam-3scale | grep 3scale-admin | awk {'print $2'})"
   ```

9. Click on `Ok, how does 3scale work?` and follow the 3Scale wizard to create an API

10. Once on your API overview page, click on `Integration` on the left, then on `Configuration`

11. Take note of the `example curl for testing` for `Staging-APIcast`

12. Trigger API Usage alerts

    | Alert to fire                        | `numRequests` | `recipients`          |
    | ------------------------------------ | ------------- | --------------------- |
    | RHOAMApiUsageLevel1ThresholdExceeded | 43            | BU and Customers      |
    | RHOAMApiUsageLevel2ThresholdExceeded | 46            | BU and Customers      |
    | RHOAMApiUsageLevel3ThresholdExceeded | 51            | BU, Customers and SRE |


    Due to the timing of the Prometheus scrape interval, getting alerts to fire on a per 1 minute basis can be challenging. To cater for this timing issue, perform the following steps:

      Prepare the API request command below, replacing `$numrequests` with the count set in the table above. Also make sure to replace `DUMMY_URL` and `DUMMY_KEY` with the values retrieved from the 3Scale console previously

      *NOTE:* Do not run the command yet!

      ```sh
      for i in {1..$numRequests}; do curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>; done
      ```

      Open the Prometheus console:

      ```sh
      open "https://$(oc get routes prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
      ```

      Add the following expression into the `Expression` field in the console:

      ```
      increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m])
      ```

      Click the `Execute` button. A `0` count should be returned.

      *NOTE:* The next steps require preparation and need to be run quickly, please read ahead before running each step

      Next, send a request to the 3Scale, making sure to replace `DUMMY_URL` and `DUMMY_KEY` with the values retrieved from the 3Scale console earlier.

      ```sh
      curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>
      ```

      Click the `Execute` button in the Prometheus console again until a count other than `0` is returned

      Now, every few seconds click the `Execute` button. As soon as the count goes back to `0` run the command prepared earlier in this section. This ensures that we send our requests in alignment with the Prometheus metrics scrape interval:

      ```sh
      for i in {1..$numRequests}; do curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>& done
      ```

      Navigate back to the prometheus console, go to `Alerts` and search for the corresponding alert name in the table above e.g. `RHOAMApiUsageLevel1ThresholdExceeded`

      After a minute or so the alert should be triggered and displayed in RED

      Repeat the above steps again for each remaining alert listed in the table above

13. Once each of the above alerts have been triggered, verify that a `FIRING` and associated `RESOLVED` email notification is sent. Check the `to` field of the email to ensure that it matches the `recipients` listed in the table above. Also ensure that the link to grafana is working as expected.

14. Verify customer facing Grafana dashboards

    ```sh
    open "https://$(oc get routes grafana-route -n redhat-rhoam-customer-monitoring-operator -o jsonpath='{.spec.host}')"
    ```

    Select the Rate Limiting dashboard

    Run the following command:

    ```sh
    for i in {1..1000}; do curl -i https://<DUMMY_URL>//?user_key=<DUMMY_KEY>& done
    ```

    _NOTE:_ The above command should eventually result in failing `429 Too Many Requests` status codes. This is to be expected. If no requests have been rejected make sure to check the current Rate Limit configuration in the `sku-limits-managed-api-service` configmap of the `redhat-rhoam-operator` namespace

    While the command is running, check that the graphs in the Grafana dashboard are updating every minute. For example, the "Last 1 Minute - Rejected" percentage should be around 90%

    See [this example](https://user-images.githubusercontent.com/4881144/99288530-07dced00-283c-11eb-9cba-906151dd7dfb.png)
