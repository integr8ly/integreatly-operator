---
environments:
  - external
estimate: 1h
tags:
  - manual-selection
---

# P01 - Measure downtime during OpenShift upgrade

## Description

Measure the downtime of the RHOAM components during a AWS Availability Zone failure to ensure that pods redistribute and
service remain available.

## Prerequisites

- Node.js installed locally
- [oc CLI v4.3](https://docs.openshift.com/container-platform/3.6/cli_reference/get_started_cli.html#installing-the-cli)
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. Make sure **nobody is using the cluster** for performing the test cases, because the RHOAM components will have a 
   downtime during this test

3. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repo and run the following command:

    ```bash
    git clone https://github.com/integr8ly/workload-web-app
    cd workload-web-app
    export GRAFANA_DASHBOARD=true
    export RHOAM=true
    make local/deploy
    ```

   See step 8 and 9, you might want to do these before killing the zone as well.

4. In terminal window #2, run the following [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/disableAz.sh) 
   to fail an AZ

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

5. Wait around 30 minutes for all terminating pod to redeploy to their new AZ's. You can use the following 
   [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/podsAz.sh) to check the pod 
   distribution.
   ```bash
   # e.g.
   ./scripts/podsAz.sh redhat-rhmi-rhsso
   Pods distribution for 'redhat-rhmi-rhsso'
   | Pod name | Availability Zone |
   | -------- | ----------------- |
   | keycloak-0 | eu-west-1c |
   | keycloak-1 | eu-west-1b |

   ``` 

6. Go to the OpenShift console, go through all the `redhat-rhmi-` prefixed namespaces and verify that all routes (Networking -> Routes) of RMOAN components are accessible

   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

7. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Upload the file to the JIRA issue. Upload the file to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

8. Open the RHOAM Grafana Console in the `redhat-rhmi-middleware-monitoring-operator` namespace

    ```bash
    echo "https://$(oc get route grafana-route -n redhat-rhmi-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
    ```

9. Select the **Workload App** dashboard

> Verify that **3scale** and **SSO** are working by checking the **Status** graph.
> Take the screenshot of the dashboard and attach it to this ticket

10. re-enable the AZ by running the same [script](https://github.com/integr8ly/integreatly-operator/blob/master/scripts/disableAz.sh) 
    as before
    ```bash
    # e.g.
    ./scripts/disableAz.sh false eu-west-1a
    ```

11. Consult the results with engineering (especially in case some components have a longer downtime than 30min 
    or are not working properly)
