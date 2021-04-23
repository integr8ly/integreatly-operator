---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-private-post-upgrade
estimate: 2h
tags:
  - per-build
---

# N02B - Upgrade RHOAM

## Description

Mesure the downtime of the RHOAM components during the RHOAM upgrade (not to be confused with the OpenShift upgrade) to ensure RHOAM can be safely upgraded.

Note: This test includes all steps to prepare the cluster before the upgrade, trigger the upgrade and collect downtime reports

Note: If [N09 test case](https://github.com/integr8ly/integreatly-operator/blob/master/test-cases/tests/upgrade/n09-verify-that-upgrades-rollout-can-be-paused.md) is scheduled to be verified for the release, it might be good to do it first (once the pre-upgrade cluster is ready for the upgrade - after the step 2)

## Prerequisites

- Node.js installed locally
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.CLUSTER_NAME.s1.devshift.org:6443
   ```

2. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repo and run the following command:

   ```
   git clone https://github.com/integr8ly/workload-web-app
   cd workload-web-app
   export GRAFANA_DASHBOARD=true RHOAM=true
   make local/deploy
   ```

   > Note: do not re-deploy if the workload-web-app is already present in the cluster.

   There should be no errors in the command output and product (3scale, SSO) URLS should not be blank. Alternatively, you can check the `Environment` tab in workload-webapp namespace in OpenShift console. See step 8 and 9, you might want to do these pre-upgrade as well.

3. Edit RHMIConfig in the `redhat-rhoam-operator` config to start the upgrade

   ```
   oc edit RHMIConfig rhmi-config -n redhat-rhoam-operator
   ```

4. Edit following fields in the **rhmi-config** and save:

   - spec.upgrade.notBeforeDays: 0
   - spec.upgrade.waitForMaintenance: `false`

   > The upgrade should start shortly. Have a look at `status.upgrade.scheduled.for`. In rare situations it might get scheduled more that 6 hours in past, in that case upgrade won't be triggered. Play with the `spec.maintenance.*` and `spec.upgrade.*` values to get it scheduled some other time.

   Use the command below to check whether the installPlan exists and is approved. The operator should approve the installPlan based on **rhoam-config**. The installPlan should not be approved manually - if the installPlan is not approved shortly, restart the rhmi-operator (delete the pod or scale down to 0 and then scale back up to 1).

   ```
   oc get installplans -n redhat-rhoam-operator
   ```

5. Poll cluster to check when the RHOAM upgrade is completed (update version to match currently tested version (e.g. `2.4.0`)):

   ```bash
   watch -n 60 " oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r .status.version | grep -q "2.x.x" && echo 'RHOAM Upgrade completed\!'"
   ```

   > This script will run every 60 seconds to check whether the RHOAM upgrade has finished
   >
   > Once it's finished, it should print out "Upgrade completed!"

6. Go to the OpenShift console, go through all the `redhat-rhoam-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHOAM components are accessible

   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

7. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --namespace redhat-rhoam-middleware-monitoring-operator --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Upload the file to the JIRA issue. Upload the file to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

8. Open the RHOAM Grafana Console in the `redhat-rhoam-middleware-monitoring-operator` namespace

```bash
echo "https://$(oc get route grafana-route -n redhat-rhoam-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
```

9. Select the **Workload App** dashboard

> Verify that **3scale** and **SSO** are working by checking the **Status** graph.
> Take the screenshot of the dashboard and attach it to this ticket
>
> Note: when testing the RHOAM upgrade the dashboard must be verified also after the upgrade and any downtime during the upgrade should be reported as issues (also make sure that the screenshot of the dashboard post-upgrade is attached to this Jira)
>
> Note: it's normal that graph will show a short downtime at the start for 3scale because the workload-web-app is usually deployed before the 3scale API is ready

10. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
