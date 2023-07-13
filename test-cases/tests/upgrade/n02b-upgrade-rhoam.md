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

Measure the downtime of the RHOAM components during the RHOAM upgrade (not to be confused with the OpenShift upgrade) to ensure RHOAM can be safely upgraded.

Note: This test includes all steps to prepare the cluster before the upgrade, trigger the upgrade and collect downtime reports

Note: If [N09 test case](https://github.com/integr8ly/integreatly-operator/blob/master/test-cases/tests/upgrade/n09-verify-that-upgrades-rollout-can-be-paused.md) is scheduled to be verified for the release, it might be good to do it first (once the pre-upgrade cluster is ready for the upgrade - after the step 2)

## Prerequisites

- Node.js installed locally
- [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
- [jq v1.6](https://github.com/stedolan/jq/releases) installed locally
- cluster with the current RHOAM GA version installed on it. Such cluster is typically created via the [addon-flow](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow) pipeline when the RC1 is not yet merged into MT (or reverted back to GA).
  - testing-idp is configured on the cluster (tick the step)
  - corrupted users are created on the cluster (tick the step)
  - sanity check is done on the cluster (tick the step)
  - workload web app is deployed on the cluster (tick the step)

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.CLUSTER_NAME.s1.devshift.org:6443
   ```

2. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --namespace redhat-rhoam-operator-observability --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Take a look if there's any downtime pre-upgrade. There might be some if the cluster was created less than 1 hour ago.

3. Use the command below to check whether the installPlan exists and is approved.

   ```
   oc get installplans -n redhat-rhoam-operator
   ```

   You can check whether the `installplan` is for the proper RC by examining the output of the command below.

   ```
   oc get installplan install-<hash> --namespace redhat-rhoam-operator -o yaml
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

6. Run the following command to generate a downtime report using the delorean cli again post-upgrade:

   ```
   ./delorean pipeline query-report --namespace redhat-rhoam-operator-observability --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
   ```

   There should be no additional downtime compared to pre-upgrade report.

> Note: the critical 3scale components that _must not_ report any downtime are `apicast-production`, `backend-worker`, and `backend-listener`. On the other hand, the non-critical 3scale components that are ok to experience short downtime (up to 2-3 minutes) are `backend-cron`, `zync-database`, `system-memcache`, `system-sphinx`.

7. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
