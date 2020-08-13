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
   export GRAFANA_DASHBOARD=true
   make local/deploy
   ```

   See step 10 and 11, you might want to do these pre-upgrade as well.

5. Edit RHMIConfig in the `redhat-rhmi-operator` config again to start the upgrade

   ```
   oc edit RHMIConfig rhmi-config -n redhat-rhmi-operator
   ```

6. Edit following fields in the **rhmi-config** and save:

   - spec.upgrade.alwaysImmediately: `true`

   > The upgrade should start automatically

7. Poll cluster to check when the RHMI upgrade is completed (update version to match currently tested version (e.g. `2.4.0`)):

   ```bash
   watch -n 60 " oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq -r .status.version | grep -q "2.x.x" && echo 'RHMI Upgrade completed\!'"
   ```

   > This script will run every 60 seconds to check whether the RHMI upgrade has finished
   >
   > Once it's finished, it should print out "Upgrade completed!"

8. Go to the OpenShift console, go through all the `redhat-rhmi-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHMI components are accessible

   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

9. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --config-file ./configurations/downtime-report-config.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Upload the file to the JIRA issue. Upload the file to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

10. Open the RHMI Grafana Console in the `redhat-rhmi-middleware-monitoring-operator` namespace

```bash
echo "https://$(oc get route grafana-route -n redhat-rhmi-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
```

11. Select the **Workload App** dashboard

> Verify that **AMQ**, **3scale** and **SSO** are working by checking the **Status** graph.
> Take the screenshot of the dashboard and attach it to this ticket
>
> Note: when testing the RHMI upgrade the dashboard must be verified also after the upgrade and any downtime during the upgrade should be reported as issues (also make sure that the screenshot of the dashboard post-upgrade is attached to this Jira)
>
> Note: it's normal that graph will show a short downtime at the start for 3scale and/or AMQ because the workload-web-app is usually deployed before the 3scale API and/or the AMQ queue is ready

12. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
