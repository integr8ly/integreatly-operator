---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.2.0
estimate: 3h
tags:
  - manual-selection
---

# P01 - Measure downtime during down az

## Description

Measure the downtime of the RHOAM components during a AWS Availability Zone failure to ensure that pods redistribute and services remain available. Time estimation is 3 hours, cluster provisioning excluded.

## Prerequisites

- Node.js installed locally
- [oc CLI v4.3](https://docs.openshift.com/container-platform/3.6/cli_reference/get_started_cli.html#installing-the-cli)
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally
- cluster with Multi AZ RHOAM installed on it
  - all the alerts should be green
  - all the tests should pass (especially [Pod Distribution test](https://github.com/integr8ly/integreatly-operator/blob/master/test/functional/multiaz_pod_distribution.go))

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. Make sure **nobody is using the cluster** for performing the test cases, because the RHOAM components will have a downtime during this test

3. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repository and run the following command (double check with `README.md`):

   ```bash
   git clone https://github.com/integr8ly/workload-web-app
   cd workload-web-app
   export GRAFANA_DASHBOARD=true
   export RHOAM=true
   make local/deploy
   ```

4. Record the pod distribution from 3scale, user-sso, rhsso, marin3r, and middleware-monitoring-operator namespaces using [podsAZ](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/podsAz.sh) script

   ```bash
   # e.g.
   ./scripts/podsAz.sh redhat-rhoam-user-sso
   Pods distribution for 'redhat-rhoam-user-sso'
   | Pod name | Availability Zone |
   | -------- | ----------------- |
   | keycloak-0 | eu-west-1c |
   | keycloak-1 | eu-west-1b |

   ```

5. Create customer-like application using customer-admin01 (or other user from dedicated-admin group)

   ```bash
   oc new-project httpbin
   oc new-app jsmadis/httpbin
   oc expose svc/httpbin
   ```

6. Manage the customer-like app by 3scale and secure it by user SSO

   Probably the simplest way to do it is to run the performance test suite. The [branch](https://gitlab.cee.redhat.com/trepel/3scale-py-testsuite/-/tree/customer-like-app) has been created to simplify this as much as possible. If you have ran the performance test suite before, you "just" have to clone the branch, update [settings.local.yaml](https://gitlab.cee.redhat.com/trepel/3scale-py-testsuite/-/blob/customer-like-app/config/settings.local.yaml) and execute the test suite

   ```bash
   pipenv run python -m pytest --performance_smoke testsuite/tests/performance/apicast/smoke/test_smoke_oidc.py
   ```

   If you haven't run the performance test suite before, see the comments in [MGDAPI-238](https://issues.redhat.com/browse/MGDAPI-238). And [Austin's doc](https://docs.google.com/document/d/1NJBUsieRkBLnN2PMAF5cpaH7uXq9mZCx1JQaT9Ruytk/edit?usp=sharing) in particular.

   If you wish to do it manually, see the comments in [MGDAPI-198](https://issues.redhat.com/browse/MGDAPI-198).

7. In terminal window #2, run the following [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/alerts-during-perf-testing.sh)

8. In terminal window #3, run a script to monitor the customer-like application

   ```bash
   #!/usr/bin/env bash

   #run this like ./nameOfThisFile.sh 2>&1 | tee nameOfThisFileOutput.txt
   while true; do
     echo `date`
     curl -isS -H 'Accept: application/json' -H "Authorization: Bearer <TOKEN>" https://<PRODUCTION-APICAST-URL> | grep HTTP | head -1
     echo "----"
     sleep 5;
   done
   ```

   If you created the customer-like application via performance test suite, connect to the machine where Hyperfoil controller is running and see the `/tmp/hyperfoil/runs/<run-number/managed_services_base_line.data/auth_oidc.csv` to get TOKEN and PRODUCTION-APICAST-URL. The `/tmp/hyperfoil` is a default value. It might be located elsewhere. If the Hyperfoil controller is deployed in OpenShift, the path is `/var/hyperfoil` instead.

   If you did it manually then you need to generate a token:

   ```bash
   curl -X POST 'https://keycloak-edge-redhat-rhoam-user-sso.apps.<YOUR-CLUSTER>.s1.devshift.org:443/auth/realms/<YOUR-REALM>/protocol/openid-connect/token' -H "Content-Type: application/x-www-form-urlencoded" --data "grant_type=password&client_id=<CLIENT-ID>&client_secret=<CLIENT-SECRET>&username=<USER>&password=<PASSWORD>" | jq -r '.access_token'
   ```

   All the required values for the command above are available in your user SSO instance.

9. In terminal window #4, run the following [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/disableAz.sh) to fail an AZ

- AZ should not host RHOAM monitoring stack (unless it spans multiple zones)
- AZ should not host customer app (unless it spans multiple zones)
- AZ should not host workload-web-app (unless it spans multiple zones)
- AZ should not host any Redis Primary node
  - typically there are three Redis instances there
  - `aws elasticache describe-replication-groups` and see where the Primaries are hosted
- AZ should host Production APIcast (one of the APIcast's pods)

  ```bash
  # check the az of the current cluster e.g.
  oc get machineset -n openshift-machine-api
  NAME                                      DESIRED   CURRENT   READY   AVAILABLE   AGE
  mw-collab-multi-45qpn-infra-eu-west-1a    1         1         1       1           70m
  mw-collab-multi-45qpn-infra-eu-west-1b    1         1         1       1           70m
  mw-collab-multi-45qpn-infra-eu-west-1c    1         1         1       1           70m
  mw-collab-multi-45qpn-worker-eu-west-1a   3         3         3       3           94m
  mw-collab-multi-45qpn-worker-eu-west-1b   3         3         3       3           94m
  mw-collab-multi-45qpn-worker-eu-west-1c   3         3         3       3           94m
  # run the script to disable the AZ e.g.
  ./scripts/disableAz.sh true eu-west-1a
  ```

10. Wait around 30 minutes for all terminating pods to redeploy to their new AZ's. You can use the following [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/podsAz.sh) to check the pod distribution.

    ```bash
    # e.g.
    ./scripts/podsAz.sh redhat-rhoam-user-sso
    Pods distribution for 'redhat-rhoam-user-sso'
    | Pod name | Availability Zone |
    | -------- | ----------------- |
    | keycloak-0 | eu-west-1c |
    | keycloak-1 | eu-west-1b |
    ```

    All the RHOAM alerts should get green eventually. You might experience OSD issues (failed `oc login...` etc) for some time (~15 minutes).

11. Go to the OpenShift console, go through all the `redhat-rhoam-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHOAM components are accessible

- If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

12. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

    ```
    cd delorean
    make build/cli
    ./delorean pipeline query-report --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
    ```

    There will be a yaml file generated in the output directory. Upload the file to the JIRA issue. Upload the file to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

13. Open the RHOAM Grafana Console in the `redhat-rhoam-middleware-monitoring-operator` namespace

    ```bash
    echo "https://$(oc get route grafana-route -n redhat-rhoam-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
    ```

14. Select the **Workload App** dashboard

- Verify that **3scale** and **SSO** are working by checking the **Status** graph.
- Take the screenshot of the dashboard and attach it to this ticket

15. Consult the results with engineering (especially in case some components have a longer downtime than 30min
    or are not working properly)

16. Re-enable the AZ by running the same [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/disableAz.sh)
    as before

    ```bash
    # e.g.
    ./scripts/disableAz.sh false eu-west-1a
    ```

17. Wait until the AZ is restored and OpenShift starts using it

```bash
# You should see a similar output to the one below when running the following oc command:
$ oc get machineset -n openshift-machine-api
NAME                                      DESIRED   CURRENT   READY   AVAILABLE   AGE
mw-collab-multi-45qpn-infra-eu-west-1a    1         1         1       1           70m
mw-collab-multi-45qpn-infra-eu-west-1b    1         1         1       1           70m
mw-collab-multi-45qpn-infra-eu-west-1c    1         1         1       1           70m
mw-collab-multi-45qpn-worker-eu-west-1a   3         3         3       3           94m
mw-collab-multi-45qpn-worker-eu-west-1b   3         3         3       3           94m
mw-collab-multi-45qpn-worker-eu-west-1c   3         3         3       3           94m
```

18. Run the automated test for checking the correct pod distribution across all AZs and make sure it passes (It might take a while until all pods are correctly redistributed, so if the test fails, try to run it again after couple of minutes. If the test keeps failing, consult the issue with engineering.)

```bash
$ go clean -testcache && MULTIAZ=true WATCH_NAMESPACE=redhat-rhoam-operator go test -v ./test/functional -run="//^F09"
```
