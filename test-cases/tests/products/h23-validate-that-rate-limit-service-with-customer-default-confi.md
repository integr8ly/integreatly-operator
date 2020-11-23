---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 1h
tags:
  - per-release
---

# H23 - Validate that Rate Limit service with customer default configuration is working as expected

## Prerequisites

- ["libra.pem" private key for ssh to PSI openstack load testing instance](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/keys/libra.pem) (follow the guide in the [README](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/README.md) to unlock the vault with git-crypt key)
- Logged in to a testing cluster as a `kubeadmin`
- Access to CSQE OCM ORG https://qaprodauth.cloud.redhat.com/beta/openshift/ (login as <kerberos-username>-csqe user)

## Description

This test case should prove that the rate limiting Redis counter correctly increases with every request made

## Steps

1. Add the correct file permission to the private key file and SSH to the load testing instance

```bash
chmod 400 /path/to/libra.pem
ssh -i /path/to/libra.pem fedora@10.0.76.255
```

2. Navigate to the 'rate-limiting' folder. Delete any existing sripts in this folder and download the script from [here](https://gist.github.com/psturc/9d7486bac0a5791d80419694721069e8) and name it `script.js`. This will set up the load testing tool.

3. Set the customer config values rate limiting:

- Ensure you are logged into the testing cluster (Multi AZ).
- Execute the command `oc get configmap sku-limits-managed-api-service -n redhat-rhoam-operator -o json | jq -r .data.rate_limit` to ensure that the values are as follows:
  "unit":"minute"
  "requests_per_unit":13860"

4.  Check that there are no additional alerts firing:

- Select the test cluster from https://qaprodauth.cloud.redhat.com/beta/openshift/
- Click the 'monitoring' tab and check the alerts.

5.  Configure an endpoint to run the load test using 3scale and promote the endpoint to production:

- Login to the API Management (3Scale) on the testing clsuter using on the the customer-adminxx idp
- Create a new endpoint using the wizard using all the default settings.
- Select 'Integration' from the menu (left) and select 'Configuration'.
- Scroll down and click 'Promote to Production' and not the Production APIcast api
- To test copy the Example curl for testing and replace 'staging' with 'production' (see exmple below)
  curl "https://api-3scale-apicast-production.apps.test-rl-psturc.0ez3.s1.devshift.org:443/?user_key=40e64b61b28a6286fda49ccef828ed80"

6. Replace the placeholder in the script (first line) with the actual APIcast production URL (in the curl command above)
7. Access Prometheus route (from your terminal on local machine)

```bash
 open "https://$(oc get routes prometheus-route -n redhat-rhoam-middleware-monitoring-operator -o jsonpath='{.spec.host}')"
```

and login as Kubeadmin.

-

8. Run the load test:

```bash
k6 run script.js
```

This will run for 10 minutes at 15,000 requests per min.

8. Go back to Prometheus web page

- Execute this query:
  `increase(ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits[1m])`
- Click on the `Graph` button
  > Ensure that the graph shows the usage is constantly at the 15,000 requests for 10 minutes

The first value (under status) is the number of passed requests.

- eg _↳ 93% — ✓ 140021 / ✗ 10195_

Divide this number by 10 and the result should be ~to the rate limit value from the config map.
eg: _1400021/10 <=> 13860_

9. Consult the results with engineering
