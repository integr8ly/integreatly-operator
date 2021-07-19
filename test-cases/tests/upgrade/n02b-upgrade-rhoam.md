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

## Prerequisites

To prepare a cluster for the upgrade testing the current GA version must be in managed-tenants repo.

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

   > Note: do not re-deploy if the workload-web-app is already present in the cluster - check if `workload-web-app` namespace exists in the cluster or not.

   There should be no errors in the command output and product (3scale, SSO) URLS should not be blank. Alternatively, you can check the `Environment` tab in workload-webapp namespace in OpenShift console. See step 7 and 8, you might want to do these pre-upgrade as well.

3. Use the command below to check whether the installPlan exists and is approved.

   ```
   oc get installplans -n redhat-rhoam-operator
   ```

   In case release candidate is service affecting, you need to approve the installPlan first.

   ```
   oc patch installplan install-<hash> --namespace redhat-rhoam-operator --type merge --patch '{"spec":{"approved":true}}'
   ```

4. Poll cluster to check when the RHOAM upgrade is completed (update version to match currently tested version (e.g. `1.8.0`)):

   ```bash
   watch -n 60 " oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r .status.version | grep -q "1.x.x" && echo 'RHOAM Upgrade completed\!'"
   ```

   > This script will run every 60 seconds to check whether the RHOAM upgrade has finished
   >
   > Once it's finished, it should print out "Upgrade completed!"

5. In OpenShift console, click on the "bell" icon in the top right corner. Go through the notifications that were generated in the time between the RHOAM upgrade started and ended.

   > Verify that there were no RHOAM-related alerts firing during the upgrade
   > If unsure whether the alert is RHOAM-related, consult it with engineering

6. Go to the OpenShift console, go through all the `redhat-rhoam-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHOAM components are accessible

   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

7. In anonymous browser window, log in to OpenShift console as a user with dedicated-admin permissions (e.g. "customer-admin01"), click on the dashboard icon on the top right corner

   > Validate that all 3 links under OpenShift Managed Services (API Management, API Management Dashboards, API Management SSO) are accessible and you can log in.

8. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --namespace redhat-rhoam-middleware-monitoring-operator --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Upload the file to the JIRA issue. Upload the file to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

> Note: the critical 3scale components that _must not_ report any downtime are `apicast-production`, `backend-worker`, and `backend-listener`. On the other hand, the non-critical 3scale components that are ok to experience short downtime (up to 2-3 minutes) are `backend-cron`, `zync-database`, `system-memcache`, `system-sphinx`.

9. Open the RHOAM Grafana Console in the `redhat-rhoam-middleware-monitoring-operator` namespace

```bash
echo "https://$(oc get route grafana-route -n redhat-rhoam-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
```

10. Select the **Workload App** dashboard

> Verify that **3scale** and **SSO** are working by checking the **Status** graph.
> Take the screenshot of the dashboard and attach it to this ticket
>
> Note: when testing the RHOAM upgrade the dashboard must be verified also after the upgrade and any downtime during the upgrade should be reported as issues (also make sure that the screenshot of the dashboard post-upgrade is attached to this Jira)
>
> Note: it's normal that graph will show a short downtime at the start for 3scale because the workload-web-app is usually deployed before the 3scale API is ready, see [MGDAPI-1266](https://issues.redhat.com/browse/MGDAPI-1266)
>
> Note: Downtime measurement might not be 100% reliable, see [MGDAPI-2333](https://issues.redhat.com/browse/MGDAPI-2333)

11. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
