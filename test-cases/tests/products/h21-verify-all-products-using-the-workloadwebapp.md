---
estimate: 30m
---

# H21 - Verify all products using the workload-web-app

## Description

The [workload-web-app](https://github.com/integr8ly/workload-web-app) will:

- Create a small AMQ Queue and verify that it can send and receive messages
- Create a user in the User SSO and verify that it can login to it
- Create a 3scale API and verify that it respond

**Note:** When testing the RHMI upgrade, the steps 1., 2. and 3. should be performed before the upgrade and the verification steps 4. and 5. after the upgrade

## Steps

1. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repository

2. Login to the cluster as **kubeadmin**

3. Deploy the **workload-web-app** to the cluster

   ```bash
    export GRAFANA_DASHBOARD=true
    make local/deploy
   ```

   > Verify tha the AMQ_QUEUE, RHSSO_SERVER_URL and THREE_SCALE_URL have been created during the deployment
   >
   > ```
   > Deploying the webapp with the following parameters:
   > AMQ_ADDRESS=amqps://...
   > AMQ_QUEUE=/...
   > RHSSO_SERVER_URL=https://...
   > THREE_SCALE_URL=https://...
   > ```

4. Open the RHMI Grafana Console in the `redhat-rhmi-middleware-monitoring-operator` namespace

   ```bash
   echo "https://$(oc get route grafana-route -n redhat-rhmi-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
   ```

5. Select the **Workload App** dashboard

   > Verify that **AMQ**, **3scale** and **SSO** are working by checking the **Status** graph.
   > Take the screenshot of the dashboard and attach it to this ticket
   >
   > Note: when testing the RHMI upgrade the dashboard must be verified also after the upgrade and any downtime during the upgrade should be reported as issues (also make sure that the screenshot of the dashboard post-upgrade is attached to this Jira)
   >
   > Note: it's normal that graph will show a short downtime at the start for 3scale and/or AMQ because the workload-web-app is usually deployed before the 3scale API and/or the AMQ queue is ready
