---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.0.0
      - 1.5.0
      - 1.7.0
      - 1.10.0
      - 1.11.0
      - 1.14.0
      - 1.19.0
      - 1.22.0
      - 1.25.0
      - 1.28.0
      - 1.31.0
      - 1.34.0
      - 1.35.0
      - 1.38.0
      - 1.39.0
      - 1.42.0
estimate: 1h
tags:
  - destructive
---

# H23 - Validate rate limit service with default customer config

## Prerequisites

- Cluster must be configured for 20M Quota
- ["libra.pem" private key for ssh to PSI openstack load testing instance](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/keys/libra.pem) (follow the guide in the [README](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/README.md) to unlock the vault with git-crypt key)
- Logged in to a testing cluster as a `kubeadmin`
- Access to CSQE OCM ORG https://qaprodauth.console.redhat.com/beta/openshift/ (login as <kerberos-username>-csqe user) to view the list of provisioned clusters.
- [JQ](https://stedolan.github.io/jq/)

## Description

This test case should prove that the rate limiting works successfully at the default custom-config level.

## Steps

**1. Patch the sre, customer & BU email address in the CR to send the email to your own email inbox.**

- Login to the testing cluster (oc login) and Set the following envs:

```bash
EMAIL_LOCAL_PART=<yourRedHatId> //eg trdoyle

EMAIL_CLUSTER_NAME=test-rate-limit
```

Note the original email addresses first. As a last step these need to be reverted back.

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o=jsonpath='{.spec.alertingEmailAddress}'
oc get rhmi rhoam -n redhat-rhoam-operator -o=jsonpath='{.spec.alertingEmailAddresses.businessUnit}'
oc get rhmi rhoam -n redhat-rhoam-operator -o=jsonpath='{.spec.alertingEmailAddresses.cssre}'
```

Run the following command to patch the CR:

```bash
oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p "{\"spec\":{\"alertingEmailAddress\":\"${EMAIL_LOCAL_PART}+${EMAIL_CLUSTER_NAME}-CUSTOMER@redhat.com\",\"alertingEmailAddresses\":{\"businessUnit\":\"${EMAIL_LOCAL_PART}+${EMAIL_CLUSTER_NAME}-BU@redhat.com\", \"cssre\":\"${EMAIL_LOCAL_PART}+${EMAIL_CLUSTER_NAME}-SRE@redhat.com\"}}}"
```

You should get the output `rhmi.integreatly.org/rhoam patched`.

You can verify that the email will be redirected by checking the status in _Alertmanager_.

- Open alert manager by following the port forwarded address using the following command (on your testing cluster)

```
oc port-forward -n redhat-rhoam-operator-observability alertmanager-rhoam-0 9093:9093
```

- Login with kube admin and select the **Status** option from the nav menu.
- Search for _email_configs_ and the _to_ value should be set to yourRedhatUsername+test-rate-limit-SRE@redhat.com
  or yourRedhatUsername+test-rate-limit-BU@redhat.com

- Patch the **rate-limit-alerts** config map to fire every 1 minute or every 10 minutes by running the following command on your test cluster

```
oc patch configmap rate-limit-alerts -n redhat-rhoam-operator -p '"data": {
  "alerts": "{\n  \"api-usage-alert-level1\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel1ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"80%\",\n      \"maxRate\": \"90%\"\n    }\n  },\n  \"api-usage-alert-level2\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel2ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"90%\",\n      \"maxRate\": \"95%\"\n    }\n  },\n  \"api-usage-alert-level3\": {\n    \"type\": \"Threshold\",\n    \"level\": \"info\",\n    \"ruleName\": \"RHOAMApiUsageLevel3ThresholdExceeded\",\n    \"period\": \"1m\",\n    \"threshold\": {\n      \"minRate\": \"95%\"\n    }\n  },\n  \"rate-limit-spike\": {\n    \"type\": \"Spike\",\n    \"level\": \"warning\",\n    \"ruleName\": \"RHOAMApiUsageOverLimit\",\n    \"period\": \"10m\"\n  }\n}"
}'
```

You can check the values **rate-limit-alerts** config map have been updated sucessfully :

```
 oc -n redhat-rhoam-operator get configmap rate-limit-alerts -o yaml
```

The **period** value for each of the alerts should be set to **1m** with the exception of for the **rate-limit-spike (RHOAMApiUsageOverLimit)** which should be set to **10m**.

An Email alert for rate limiting should be in your inbox when rate-limits are hit/exceeded during testing (step 9) .

**2. Get the private-key to access the rate-limit testing instance**

You will need the private key file named **'libra.pem'** from the vault repo to access the rate-limit-testing instance machine.

To use this key you will first need to update your local instance key-file with the correct file permission as follows:

```bash
chmod 400 /path/to/vault/libra.pem
```

Then SSH to the load testing instance (using the key-file):

```bash
ssh -i /path/to/libra.pem fedora@10.0.77.22
```

you should now be logged into [fedora@rate-limit-testing ~]. If not, start the [instance](https://rhos-d.infra.prod.upshift.rdu2.redhat.com/dashboard/project/instances/7e7415f4-d702-4b2e-b0e9-13975e220b50/)

list the folders - you should see a 'rate-limit-testing' folder.

**3. Navigate to the 'rate-limiting' folder.**

Delete any existing scripts.js files in this folder

Run the command (below)
to download the script from [here](https://gist.githubusercontent.com/trepel/48186acce76190d4519e876c4db280f1/raw/b9a4b93938829212c53223dbc0eb5582958f2273/script.js -O script.js) and rename the file to script.js. The downloaded script will be used to set up the load testing tool.

```bash
wget https://gist.githubusercontent.com/trepel/48186acce76190d4519e876c4db280f1/raw/b9a4b93938829212c53223dbc0eb5582958f2273/script.js -O script.js
```

The script.js file should now be downloaded.

**4. Set the customer config rate limiting values :**

- Ensure you are logged into the testing cluster.
- Execute the following command

```bash
oc get configmap ratelimit-config -n redhat-rhoam-marin3r -o json | jq -r .data
```

Ensure that the values are as follows (should be so if 20M Quota is used):

_"seconds":60_

_"max_value":13889"_

If the values are **not** correct make sure the 20M Quota is used. If not, change the Quota via editing the RHOAM addon config via OCM.

1. Login to OCM with provided token

```bash
ocm login --url=https://api.stage.openshift.com/ --token=<YOUR_TOKEN>
```

2. Set cluster name variable

```bash
CLUSTER_NAME="<CLUSTER_NAME>"
```

3. Get cluster id and assign it to a variable

```bash
CLUSTER_ID=$(ocm get clusters --parameter search="name like '%$CLUSTER_NAME%'" | jq -r '.items[].id')
```

4. Change quota to 20 million

```bash
ocm patch /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service --body=<<EOF
{
   "parameters":{
      "items":[
         {
            "id":"addon-managed-api-service",
            "value":"200"
         }
      ]
   }
}
EOF
```

5. Check addon parameters updated successfully

```bash
ocm get /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service
```

**5. Check that there are no critical alerts firing:**

- Login to the openshift console as kubeadmin.
- Navigate to Observe > Alerting.
- Check that there are no critical alerts firing.

**6. Configure an endpoint to run the load test using 3scale and promote the endpoint to production:**

- Login to the API Management (3Scale) on the testing cluster using the testing idp **customer-adminxx**
- If the 3scale wizard doesn't show up after accessing the 3scale webpage, update the webpage URL to "https://\<YOUR-3SCALE-ROUTE\>/p/admin/onboarding/wizard" to access the 3scale wizard
- Click on `Ok, how does 3scale work?` and `Got it! Lets add my API`
- On the page for adding a backend, you need to add a custom one. Run the following commands:

```bash
oc new-project httpbin && \
oc new-app quay.io/trepel/httpbin && \
oc scale deployment/httpbin --replicas=6 && \
printf "\n3scale Backend Base URL: http://$(oc get svc -n httpbin --no-headers | awk '{print $3}'):8080\n"
```

- Copy the `3scale Backend Base URL` to clipboard and add it to Base URL field in the 3scale wizard
- Finish the 3scale wizard
- When completed, you will be on the Overview page. Select 'Integration' from the menu listings on the left and select 'Configuration'.
- Scroll down and click 'Promote to Production' (if you didn't go through the wizard, you have to promote it to staging first)
- take a copy of the Staging APIcast `Example curl for testing` and extract the URL (within the double quotes) and replace the word **'staging'** with **'production'** in the URL (see example below)

EXAMPLE:

Change (original)

```
https://api-3scale-apicast-staging.apps.mgdapi-84-trdoy.ro2p.s1.devshift.org:443/?user_key=7e9c1ef1c9c156af05fa7894f4a3529f
```

To

```
https://api-3scale-apicast-production.apps.mgdapi-84-trdoy.ro2p.s1.devshift.org:443/?user_key=7e9c1ef1c9c156af05fa7894f4a3529f
```

Note: this is required because there are 3scale internal limitations on load that Staging Apicast is allowed to process.

This URL will get added to the script.js (next step).

**7. Edit the script.js**
Use replace the placeholder in the script **const url** value (first line) with the actual APIcast production URL from the curl command above (including the user_key) and save the file.

Example:

```
const url = "https://api-3scale-apicast-production.apps.mgdapi-84-trdoy.ro2p.s1.devshift.org:443/?user_key=7e9c1ef1c9c156af05fa7894f4a3529f"
```

**8. Port forward the Prometheus Route (from your terminal on local machine)**

```bash
oc port-forward -n redhat-rhoam-operator-observability prometheus-rhoam-0 9090:9090
```

follow the port forwarded url and login to Prometheus as _Kubeadmin_.

**9. Run the load test:**

To execute the rate-limit test, run the following command from the **fedora@rate-limit-testing** terminal

```bash
k6 run script.js
```

This will run for 10 minutes at 15,000 requests per min.

**10. Go back to Prometheus web page**

- Execute this query:
  `sum(increase(authorized_calls[1m])) + sum(increase(limited_calls[1m]))`
- Click on the `Graph` button

  > Ensure that the graph shows the usage reaches 15,000 requests and remains at the level for 10 minutes (until the test completes).

- You may need to refresh the graph to see full results over time.
- Whent testing is complete, view the output in the terminal. The first value (under status) is the number of passed requests.
- If experiencing issues with the query, go to Customer Grafana, Ratelimit dashboard and use the query that is being used there

**Example:**
_↳ 93% — ✓ 140021 / ✗ 10195_\*

Divide this number (first value under 'status') by 10 and the result should be (approximately) equal to the rate limit value from the config map.
eg: _140021/10 <=> 13860_
We would expect the result to be (close to) ≈ 14,000

**11. Check that only one email alert has been generated**
(Threshold is 1 in every 24 hours).
Email alerts for rate limiting should be in your inbox. (as configured in step 2).
The relevant alerts for Rate Limiting are:

- _RHOAMApiUsageLevel3ThresholdExceeded_
- _RHOAMApiUsageOverLimit_

**12. Revert the email addresses back**

Use the command from Step #1, just use the original email addresses.

**13. Shut Off the rate-limit-testing Instance**

Navigate to [instance details](https://rhos-d.infra.prod.upshift.rdu2.redhat.com/dashboard/project/instances/7e7415f4-d702-4b2e-b0e9-13975e220b50/) and `shut off` it.
