---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 30m
tags:
  - per-build
---

# H21B - Verify all products using the workload-web-app

## Description

The [workload-web-app](https://github.com/integr8ly/workload-web-app) will:

- Create a user in the User SSO and verify that it can login to it
- Create a 3scale API and verify that it respond

**Note:** When testing the RHOAM upgrade, the steps 1., 2. and 3. should be performed before the upgrade and the verification steps 4. and 5. before and after the upgrade

## Steps

1. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repository

2. Login to the cluster as **kubeadmin**

3. Deploy the **workload-web-app** to the cluster.

   IMPORTANT. Make sure that you don't run `make local/deploy` again, as this will break the monitoring dashboard. If **workload-web-app** namespace exists in the cluster, the workload-web-app has already been deployed.

   ```bash
    export GRAFANA_DASHBOARD=true RHOAM=true
    make local/deploy
   ```

   > Verify that the RHSSO_SERVER_URL and THREE_SCALE_URL have been created during the deployment
   >
   > ```
   > Deploying the webapp with the following parameters:
   > RHSSO_SERVER_URL=https://...
   > THREE_SCALE_URL=https://...
   > ```

4. Open the RHOAM Grafana Console in the `redhat-rhoam-observability` namespace

   ```bash
   echo "https://$(oc get route grafana-route -n redhat-rhoam-observability -o=jsonpath='{.spec.host}')"
   ```

5. Select the **Workload App** dashboard

   > Verify that **3scale** and **SSO** are working by checking the **Status** graph.
   > Take the screenshot of the dashboard and attach it to this ticket
   >
   > Note: when testing the RHOAM upgrade the dashboard must be verified also after the upgrade and any downtime during the upgrade should be reported as issues (also make sure that the screenshot of the dashboard post-upgrade is attached to this Jira)
   >
   > Note: it's normal that graph will show a short downtime at the start for 3scale because the workload-web-app is usually deployed before the 3scale API is ready, see [MGDAPI-1266](https://issues.redhat.com/browse/MGDAPI-1266)
