---
automation:
  - INTLY-7413
environments:
  - osd-post-upgrade
estimate: 15m
tags:
  - per-build
---

# C05 - Verify alerts are exposed and no critical alerts have fired

## Prerequisites

- access to `integreatly-notifications@redhat.com` email list
- workload webapp should be running on cluster and have been deployed shortly after cluster was provisioned

## Description

Verify that the RHMI alerts are exposed via Telementry to cloud.redhat.com and that no critical RHMI alerts have fired during lifespan of cluster.

This should be one of the last testcases performed on a cluster to allow for maximum burn-in time on cluster.

Testcase should not be performed on a cluster that has been used for destructive testing.

## Steps

1. Login via `oc` as **kubeadmin**

2. Confirm the e-mail address where the alert notifications are sent, it should be `cloud-services-qe-reporting@redhat.com`. If this test is being carried out on cluster post upgrade the e-mail address will be an Intgreatly email account. Access can be granted by reaching out to a member of the Integreatly Engineering team.

   ```bash
   CSV_NAME=$(oc -n redhat-rhmi-operator get csv | grep integreatly-operator | awk '{print $1}')
   oc -n redhat-rhmi-operator get csv $CSV_NAME -o json | jq '.spec.install.spec.deployments[] | select(.name=="rhmi-operator") | .spec.template.spec.containers[] | select(.name=="rhmi-operator") | .env[] | select(.name=="ALERTING_EMAIL_ADDRESS")'
   ```

3. Check the inbox of the e-mail address and check if there are any alert notifications that are not related to testing. This can be acheived by subscribing to cloud-services-qe-reporting@redhat.com here: https://post-office.corp.redhat.com/mailman/listinfo/cloud-services-qe-reporting or alternatively you can view the archives without subscription here: http://post-office.corp.redhat.com/archives/cloud-services-qe-reporting/

4. Check there are no currently firing alerts. From the cluster manager console on https://qaprodauth.cloud.redhat.com/beta/openshift/

   - Select the test cluster
   - Select the Monitoring tab
   - Expand Alerts firing from the menu

   > The only RHMI alert here should be DeadMansSwitch.
   >
   > Note: there may be other alerts from the Openshift firing, however for the purposses of this test, it only fails if RHMI alerts are firing here.

5. Check no unexpected alert email notifications were received. Check this when cluster is more than a few hours old, at least 1 day old, and before cluster is deprovisioned.
   > If any critical alerts fired during any of these periods:
   >
   > 1. Take screenshots showing the time the alerts fired and when they were resolved
   > 2. Create a followup bug JIRA and inform release coordinators. Example JIRA: https://issues.redhat.com/browse/INTLY-9443
   > 3. Request that cluster lifespan be extended to allow time for cluster to be investigated (ask release coordinator).
