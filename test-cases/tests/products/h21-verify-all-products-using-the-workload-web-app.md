---
estimate: 30m
---

# Verify all products using the workload-web-app

## Description

The [workload-web-app](https://github.com/integr8ly/workload-web-app) will:

- Create a small AMQ Queue and verify that is it can send and receive messages
- Create a user in the User SSO and verify that it can login to it
- Create a 3scale API and verify that it respond

## Steps

1. Clone the [workload-web-app](https://github.com/integr8ly/workload-web-app) repository

2. Deploy the **workload-web-app** to the cluster

   ```bash
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

3. Open the RHMI Grafana Console in the `redhat-rhmi-middleware-monitoring-operator` namespace

   ```bash
   echo "https://$(oc get route grafana-route -n redhat-rhmi-middleware-monitoring-operator -o=jsonpath='{.spec.host}')"
   ```

4. Select the **Workload App** dashboard

   > Verify that **AMQ**, **3scale** and **SSO** are working by checking the **Status** graph
   >
   > Note: is normal that there could be some downtime because the workload-web-app could had start testing before the some of the endpoints where ready
