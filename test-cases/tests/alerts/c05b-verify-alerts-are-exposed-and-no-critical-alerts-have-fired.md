---
automation:
  - INTLY-7413
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
estimate: 15m
tags:
  - per-build
---

# C05B - Verify alerts are exposed and no critical alerts have fired

## Prerequisites

- access to `cloud-services-qe-reporting@redhat.com` email list
- [workload webapp](https://github.com/integr8ly/workload-web-app) should be running on cluster and have been deployed shortly after cluster was provisioned

## Description

> Note: double-check that the workload webapp is not already deployed before attempting to deploy it.
> Note: some alerts might fire due to automated tests and destructive test-cases, these can be safely ignored.

Verify that the RHOAM alerts are exposed via Telementry to cloud.redhat.com and that no critical RHOAM alerts have fired during lifespan of cluster.

This should be one of the last testcases performed on a cluster to allow for maximum burn-in time on cluster.

Testcase should not be performed on a cluster that has been used for destructive testing.

## Steps

1. Login via `oc` as **kubeadmin**

2. Confirm the e-mail address where the alert notifications are sent, it should be `cloud-services-qe-reporting@redhat.com`.

   ```bash
   oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r .spec.alertingEmailAddress
   ```

3. Check the inbox of the e-mail address and check if there are any alert notifications that are not related to testing. This can be acheived by subscribing to cloud-services-qe-reporting@redhat.com here: https://post-office.corp.redhat.com/mailman/listinfo/cloud-services-qe-reporting or alternatively you can view the archives without subscription here: http://post-office.corp.redhat.com/archives/cloud-services-qe-reporting/

4. Check there are no currently firing alerts.

   - Access the prometheus pod on the `redhat-rhoam-operator-observability` namespace with:

     - `oc exec -n redhat-rhoam-operator-observability prometheus-rhoam-0 -- wget -qO- --header='Accept: application/json' --no-check-certificate http://localhost:9090/api/v1/alerts | jq -r '.data.alerts[] | [.labels.alertname, .state, .activeAt] | @tsv'`

   - Confirm that only one alert DeadMansSwitch is firing

   > The only RHOAM alert here should be DeadMansSwitch.

5. Check no unexpected alert email notifications were received. Check this when cluster is more than a few hours old, at least 1 day old, and before cluster is deprovisioned.

   > If any critical alerts fired during any of these periods:
   >
   > 1. Take note of the time the alerts fired and when they were resolved
   > 2. Create a followup bug JIRA and inform release coordinators. Example JIRA: https://issues.redhat.com/browse/INTLY-9443
   > 3. Request that cluster lifespan be extended to allow time for cluster to be investigated (ask release coordinator).

6. Open the RHOAM Grafana Console in the `redhat-rhoam-customer-monitoring-operator` namespace

```bash
echo "https://$(oc get route grafana-route -n redhat-rhoam-customer-monitoring-operator -o=jsonpath='{.spec.host}')"
```

7. Select the **Workload App** dashboard

> Verify that **3scale** and **SSO** are working by checking the **Status** graph.
> Make sure the proper time interval is selected (you can ignore downtimes during automated tests and destructive test-cases).
> Short initial 3scale downtime is expected, it is a [known issue](https://issues.redhat.com/browse/MGDAPI-1266)
> Downtime measurement might not be 100% reliable, see [MGDAPI-2333](https://issues.redhat.com/browse/MGDAPI-2333)
