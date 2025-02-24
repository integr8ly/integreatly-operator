---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.0.0
      - 1.5.0
      - 1.8.0
      - 1.11.0
      - 1.13.0
      - 1.16.0
estimate: 4h
tags:
  - destructive
  - manual-selection
---

# K04 - Run performance test against RHOAM (3scale + user SSO)

## Description

Run performance tests against 3scale + user SSO to validate the advertised load. Time estimation does not include the cluster provisioning.

## Prerequisites

- [oc CLI v4.3](https://docs.openshift.com/container-platform/3.6/cli_reference/get_started_cli.html#installing-the-cli)
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally
- Python environment (python 3.x, pip, pipenv)
- RHOAM cluster ready
  - all the alerts should be green
  - all the automated tests should pass

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. Make sure **nobody is using the cluster** for performing the test case so that performance test results are not affected by any unrelated workload.

3. Create customer-like application using `customer-admin01` (or other user from dedicated-admin group)

   ```bash
   oc new-project httpbin
   oc new-app jsmadis/httpbin
   ```

4. In terminal window #2, run the [alerts-check.sh](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/alerts-check.sh) script to capture alerts pending/firing during performance test run.

5. Configure rate limiting to allow for enough requests per minute.

- go to redhat-rhmi-operator namespace
- see the `sku-limits-managed-api-service` Config Map
- edit the value of `requests_per_unit`
- wait for redeploy of ratelimit pods in redhat-rhmi-marin3r namespace

  - should be done automatically in a few minutes

    Note: This not possible for installations via addon-flow since Hive would revert your modifications to whatever
    is set in Managed Tenants repository in [sku-limits.yaml.j2](https://gitlab.cee.redhat.com/service/managed-tenants/-/blob/master/addons/managed-api-service/metadata/stage/sku-limits.yaml.j2) file.

6. In terminal window #2, run the following [script for alert watching](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/alerts-check.sh)

7. Run the performance test suite

   The way to do it is described in [MGDAPI-238](https://issues.redhat.com/browse/MGDAPI-238) and in [Austin's doc](https://docs.google.com/document/d/1NJBUsieRkBLnN2PMAF5cpaH7uXq9mZCx1JQaT9Ruytk/edit?usp=sharing). Use [trepel fork](https://gitlab.cee.redhat.com/trepel/3scale-py-testsuite/-/tree/performance_tests). To validate the advertised load use [rhsso_tokens](https://gitlab.cee.redhat.com/trepel/3scale-py-testsuite/-/blob/performance_tests/testsuite/tests/performance/apicast/smoke/template_rhsso_tokens.hf.yaml) benchmark. It is set to have 10% of login flow requests. You will need to change `maxSessions` (~6000), `usersPerSec` (~25 to validate 20M load), `duration`, `maxDuration`, and [http.sharedConnections](https://gitlab.cee.redhat.com/trepel/3scale-py-testsuite/-/blob/performance_tests/testsuite/tests/performance/apicast/smoke/test_smoke_rhsso_tokens.py#L54-55) (~1200).

- create a `perf-test-start-time.txt` file as described in [capture_resource_metrics script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/capture_resource_metrics.sh).
  - the actual performance test run doesn't start immediately, the performance test suite creates various 3scale (Product, Backend, Application, Application Plan) and SSO (realm, client, users) entities first
  - best to track the log of Hyperfoil controller to get the exact time the when the `rampUp` phase starts
  - create the file in the directory where the script resides

8. Create a `perf-test-end-time.txt` for [capture_resource_metrics script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/capture_resource_metrics.sh)

- create the file in the directory where the script resides
- to get the exact time track the Hyperfoil controller log

9. Collect the data about the performance test run

- from [alerts-check.sh](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/alerts-check.sh)
- from Hyperfoil
  - install Hyperfoil locally
  - bin/cli.sh
  - connect <hyperfoil-url-without-protocol> -p <port-8090-is-default>
  - runs # to see all the runs
  - status <your-run-name>
  - stats <your-run-name>
  - export -f json -d . <your-run-name> # to export the data
  - use [report tool](https://github.com/Hyperfoil/report) to generate the HTML out of the exported data
- review alerts based on the outcome of [the script for alert watching](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/alerts-check.sh)
  - there should be no firings for 20M benchmark
- eye review of various Grafana Dashboards, see [this guide](https://docs.google.com/document/d/1KznoB-we73lGUViJApVHyBoIgh3xpgyak6ODAEAHbwk/edit?usp=sharing) on how to do it
- use [capture_resource_metrics script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/capture_resource_metrics.sh) to get the data
- add a new column about the run to the [Load Testing](https://docs.google.com/spreadsheets/d/1v_bZIk8B_thZi93hGBNiOOnbSix0gmpwy3LV_s4WFPw/edit?usp=sharing) spreadsheet
  - fill in the first few rows with all the relevant information about the benchmark used
  - rest of the rows should be filled in based on the [capture_resource_metrics script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/capture_resource_metrics.sh)
  - add any additional info (e.g. links to Hyperfoil report, Grafana Dashboard snapshots etc)

10. Analyse the results

- compare with the previous runs

11. Attach the spreadsheet to the JIRA ticket

- store the Hyperfoil report in Google Drive
- store the Grafana Dashboard snapshot(s) there too if needed
