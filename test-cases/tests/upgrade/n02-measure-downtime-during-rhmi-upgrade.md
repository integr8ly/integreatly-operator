---
estimate: 2h
---

# N02 - Measure downtime during RHMI upgrade

## Description

Mesure the downtime of the RHMI components during the RHMI upgrade (not to be confused with the OpenShift upgrade) to ensure RHMI can be safely upgraded.

**Note:** The steps 1., 2. and 3. should be executed before the upgrade, and the rest of the steps after the upgrade

## Prerequisites

- Node.js installed locally
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repo and run the following command:

   ```
   git clone https://github.com/integr8ly/workload-web-app
   cd workload-web-app
   make local/deploy
   ```

3. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the `measure-downtime.js` script:

   ```bash
   git clone https://github.com/integr8ly/delorean
   cd delorean/scripts/ocm
   node measure-downtime.js
   ```

4. Trigger the upgrade of RHMI on the cluster following the N04 test case

5. In a separate terminal, login to the ocm staging environment and get the ID of the cluster that is going to be upgraded:

   ```bash
   # Get the token at https://qaprodauth.cloud.redhat.com/openshift/token
   ocm login --url=https://api.stage.openshift.com --token=<YOUR-TOKEN>
   CLUSTER_ID=$(ocm cluster list | grep <CLUSTER-NAME> | awk '{print $1}')
   ```

6. Poll cluster to check when the RHMI upgrade is completed (update version to match currently tested version (e.g. `2.4.0`)):

   ```bash
   watch -n 60 " oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq -r .status.version | grep -q "2.x.x" && echo 'RHMI Upgrade completed\!'"
   ```

   > This script will run every 60 seconds to check whether the RHMI upgrade has finished
   >
   > Once it's finished, it should print out "Upgrade completed!"

7. Go to the OpenShift console, go through all the `redhat-rhmi-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHMI components are accessible

   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

8. Run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --config-file ./configurations/downtime-report.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Upload the file to the JIRA issue.

9. Terminate the process for measuring the downtime of components in terminal window #1

   > It takes couple of seconds until all results are collected
   >
   > The results will be written down to the file `downtime.json`

10. Upload that file to the JIRA ticket

11. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
