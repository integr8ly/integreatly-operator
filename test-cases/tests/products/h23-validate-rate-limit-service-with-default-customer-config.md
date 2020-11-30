---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 1h
tags:
  - per-release
---

# H23 - Validate rate limit service with default customer config

## Prerequisites

- ["libra.pem" private key for ssh to PSI openstack load testing instance](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/keys/libra.pem) (follow the guide in the [README](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/README.md) to unlock the vault with git-crypt key)
- Logged in to a testing cluster as a `kubeadmin`
- Access to CSQE OCM ORG https://qaprodauth.cloud.redhat.com/beta/openshift/ (login as <kerberos-username>-csqe user) to view the list of provisioned clusters.

## Description

This test case should prove that the rate limiting works successfully at the default custom-config level.

## Steps

**1. Patch the sre, customer & BU email address in the CR to send the email to your own email inbox.**

- Login to the testing cluster (oc login) and Set the following envs:

```bash
EMAIL_LOCAL_PART=<yourRedHatId> //eg trdoyle

EMAIL_CLUSTER_NAME=test-rate-limit
```

Run the following command to patch the CR:

```bash
oc patch rhmi rhoam -n redhat-rhoam-operator --type merge -p "{\"spec\":{\"alertingEmailAddress\":\"${EMAIL_LOCAL_PART}+${EMAIL_CLUSTER_NAME}-CUSTOMER@redhat.com\",\"alertingEmailAddresses\":{\"businessUnit\":\"${EMAIL_LOCAL_PART}+${EMAIL_CLUSTER_NAME}-BU@redhat.com\", \"cssre\":\"${EMAIL_LOCAL_PART}+${EMAIL_CLUSTER_NAME}-SRE@redhat.com\"}}}"
```

You should get the output `rhmi.integreatly.org/rhoam patched`.

You can verify that the email will be redirected by checking the status in _Alertmanager_.

- Open alert manager with the command (on your testing cluster)

```
open "https://$(oc get route alertmanager-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
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
ssh -i /path/to/libra.pem fedora@10.0.76.255
```

you should now be logged into [fedora@rate-limit-testing ~]

list the folders - you should see a 'rate-limit-testing' folder.

**3. Navigate to the 'rate-limiting' folder.**

Delete any existing scripts.js files in this folder

Run the command (below)
to download the script from [here](https://gist.github.com/psturc/9d7486bac0a5791d80419694721069e8) and rename the file to script.js. The downloaded script will be used to set up the load testing tool.

```bash
wget https://gist.githubusercontent.com/psturc/9d7486bac0a5791d80419694721069e8/raw/e3d6baca6a1f9cdd3be7c82f3313de5ecad1de75/script.js -O script.js
```

The script.js file should now be downloaded.

**4. Set the customer config rate limiting values :**

- Ensure you are logged into the testing cluster.
- Execute the following command

```bash
oc get configmap sku-limits-managed-api-service -n redhat-rhoam-operator -o json | jq -r .data.rate_limit
```

Ensure that the values are as follows:

_"unit":"minute"_

_"requests_per_unit":13860"_

If the values are **not** correct, edit the config map using the command (Otherwise move on to step 5.)

`oc edit configmap sku-limits-managed-api-service -n redhat-rhoam-operator -o json`

change the values to match the following

```bash
{"RHOAM SERVICE SKU":{
  "unit":"minute",
  "requests_per_unit":13860,
  "soft_daily_limits":
  [5000000,10000000,15000000]}}
```

**5. Check that there are no critical alerts firing:**

- Select the test cluster from https://qaprodauth.cloud.redhat.com/beta/openshift/
- Click the 'monitoring' tab and check there are no critical alerts firing.

**6. Configure an endpoint to run the load test using 3scale and promote the endpoint to production:**

- Login to the API Management (3Scale) on the testing cluster using on the the testing idp **customer-adminxx**
- Create a new endpoint by clicking though all the wizard options using the default settings.
- When completed, you will be on the Overview page. Select 'Integration' from the menu listings on the left and select 'Configuration'.
- Scroll down and click 'Promote to Production'
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

This URL will get added to the script.js (next step).

**7. Edit the script.js**
Use replace the placeholder in the script **const url** value (first line) with the actual APIcast production URL from the curl command above (including the user_key) and save the file.

Example:

```
const url = "https://api-3scale-apicast-production.apps.mgdapi-84-trdoy.ro2p.s1.devshift.org:443/?user_key=7e9c1ef1c9c156af05fa7894f4a3529f"
```

**8. Access Prometheus route (from your terminal on local machine)**

```bash
 open "https://$(oc get routes prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
```

and login to Prometheus as _Kubeadmin_.

**9. Run the load test:**

To execute the rate-limit test, run the following command from the **fedora@rate-limit-testing** terminal

```bash
k6 run script.js
```

This will run for 10 minutes at 15,000 requests per min.

**10. Go back to Prometheus web page**

- Execute this query:
  `increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m])`
- Click on the `Graph` button

  > Ensure that the graph shows the usage reaches 15,000 requests and remains at the level for 10 minutes (until the test completes).

- You may need to refresh the graph to see full results over time.
- Whent testing is complete, view the output in the terminal. The first value (under status) is the number of passed requests.

**Example:**
_↳ 93% — ✓ 140021 / ✗ 10195_\*

Divide this number (first value under 'status') by 10 and the result should be (approximately) equal to the rate limit value from the config map.
eg: _1400021/10 <=> 13860_
We would expect the result to be (close to) ≈ 14,000

**11. Check that only one email alert has been generated**
(Threshold is 1 in every 24 hours).
Email alerts for rate limiting should be in your inbox. (as configured in step 2).
The relevant alerts for Rate Limiting are:
**RHOAMApiUsageLevel3ThresholdExceeded**

&

**RHOAMApiUsageOverLimit**
