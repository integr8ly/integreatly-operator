---
environments:
  - osd-post-upgrade
  - osd-private-post-upgrade
estimate: 2h
tags:
  - per-build
---

# N02 - Upgrade RHMI

## Description

Mesure the downtime of the RHMI components during the RHMI upgrade (not to be confused with the OpenShift upgrade) to ensure RHMI can be safely upgraded.

Note: This test includes all steps to prepare the cluster before the upgrade, trigger the upgrade and collect downtime reports

## Prerequisites

- Node.js installed locally
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.CLUSTER_NAME.s1.devshift.org:6443
   ```

2. Edit RHMIConfig in the `redhat-rhmi-operator` config to prevent the available upgrade from being applied

   ```
   oc edit RHMIConfig rhmi-config -n redhat-rhmi-operator
   ```

3. Edit following fields in the **rhmi-config** and save:

   - spec.upgrade.alwaysImmediately: `false`
   - spec.upgrade.duringNextMaintenance: `false`
   - spec.maintenance.applyOn: `""`

   > The upgrade should not be applied automatically with the settings described above

4. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repo and run the following command:

   ```
   git clone https://github.com/integr8ly/workload-web-app
   cd workload-web-app
   make local/deploy
   ```

5. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the `measure-downtime.js` script:

   ```bash
   git clone https://github.com/integr8ly/delorean
   cd delorean/scripts/ocm
   node measure-downtime.js
   ```

6. Edit RHMIConfig in the `redhat-rhmi-operator` config again to start the upgrade

   ```
   oc edit RHMIConfig rhmi-config -n redhat-rhmi-operator
   ```

7. Edit following fields in the **rhmi-config** and save:

   - spec.upgrade.alwaysImmediately: `true`

   > The upgrade should start automatically

8. In a separate terminal, login to the ocm staging environment and get the ID of the cluster that is going to be upgraded:

   ```bash
   # Get the token at https://qaprodauth.cloud.redhat.com/openshift/token
   ocm login --url=https://api.stage.openshift.com --token=<YOUR-TOKEN>
   CLUSTER_ID=$(ocm cluster list | grep <CLUSTER-NAME> | awk '{print $1}')
   ```

9. Poll cluster to check when the RHMI upgrade is completed (update version to match currently tested version (e.g. `2.4.0`)):

   ```bash
   watch -n 60 " oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq -r .status.version | grep -q "2.x.x" && echo 'RHMI Upgrade completed\!'"
   ```

   > This script will run every 60 seconds to check whether the RHMI upgrade has finished
   >
   > Once it's finished, it should print out "Upgrade completed!"

10. Go to the OpenShift console, go through all the `redhat-rhmi-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHMI components are accessible

    > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

11. Run the following command to generate a downtime report using the delorean cli:

    ```
    cd delorean
    make build/cli
    ./delorean pipeline query-report --config-file ./configurations/downtime-report.yaml -o <output_dir>
    ```

    There will be a yaml file generated in the output directory. Upload the file to the JIRA issue.

12. Terminate the process for measuring the downtime of components in terminal window #1

    > It takes couple of seconds until all results are collected
    >
    > The results will be written down to the file `downtime.json`

13. Upload that file to the JIRA ticket

14. Upload all reports to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

15. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
