---
products:
  - name: rhoam
    environments:
      - external
estimate: 2h
tags:
  - per-release
---

# N01B - Measure downtime during OpenShift upgrade

## Description

Mesure the downtime of the RHOAM components during the OpenShift upgrade (not to be confused with the RHOAM upgrade) to ensure OpenShift can be safely upgraded.

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

2. Make sure **nobody is using the cluster** for performing the test cases, because the RHOAM components will have a downtime during the upgrade

3. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repo and run the following command:

   ```
   git clone https://github.com/integr8ly/workload-web-app
   cd workload-web-app
   export GRAFANA_DASHBOARD=true RHOAM=true
   make local/deploy
   ```

   > Note: do not re-deploy if the workload-web-app is already present in the cluster - check if `workload-web-app namespace exists or not.

   See step 9 and 10, you might want to do these pre-upgrade as well.

4. In terminal window #2, run the following command to trigger the OpenShift upgrade

   ```bash
   oc adm upgrade --to-latest=true
   ```

   > You should see the message saying the upgrade of the OpenShift cluster is triggered

   - In case of upgrade between minor versions you might need to change the channel first
     - `oc patch clusterversion/version -p '{"spec":{"channel":"stable-4.y"}}' --type=merge`
     - see the [Knowledgebase article](https://access.redhat.com/solutions/4606811) for details

5. Ask QE team to login to the ocm staging environment and get the ID of the cluster that is going to be upgraded:

   ```bash
   # Get the token at https://qaprodauth.cloud.redhat.com/openshift/token
   ocm login --url=https://api.stage.openshift.com --token=YOUR_TOKEN
   CLUSTER_ID=$(ocm cluster list | grep <CLUSTER-NAME> | awk '{print $1}')
   ```

6. Run this command to wait for the OpenShift upgrade to complete:

   ```bash
   watch -n 60 "ocm get cluster $CLUSTER_ID | jq -r .metrics.upgrade.state | grep -q completed && echo 'Upgrade completed\!'"
   ```

   > This script will run every 60 seconds to check whether the OpenShift upgrade has finished
   >
   > Once it's finished, it should print out "Upgrade completed!" (it could take ~1 hour)

7. Go to the OpenShift console, go through all the `redhat-rhoam-` prefixed namespaces and verify that all routes (Networking -> Routes) of RHOAM components are accessible

   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.

8. Clone [delorean](https://github.com/integr8ly/delorean) repo and run the following command to generate a downtime report using the delorean cli:

   ```
   cd delorean
   make build/cli
   ./delorean pipeline query-report --namespace redhat-rhoam-observability --config-file ./configurations/downtime-report-config-rhoam.yaml -o <output_dir>
   ```

   There will be a yaml file generated in the output directory. Upload the file to the JIRA issue. Upload the file to this [google drive folder](https://drive.google.com/drive/folders/10Gn8fMiZGgW_34kHlC2n1qigdfJytCpx?usp=sharing)

> Note: the critical 3scale components that _must not_ report any downtime are `apicast-production`, `backend-worker`, and `backend-listener`. On the other hand, the non-critical 3scale components that are ok to experience short downtime (up to 2-3 minutes) are `backend-cron`, `zync-database`, `system-memcache`, `system-sphinx`.

9. Open the RHOAM Grafana Console in the `redhat-rhoam-observability` namespace

```bash
echo "https://$(oc get route grafana-route -n redhat-rhoam-observability -o=jsonpath='{.spec.host}')"
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
