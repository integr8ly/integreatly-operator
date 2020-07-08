---
estimate: 15m
automation_jiras:
  - INTLY-7413
---

# C05 - Verify that the alerts are exposed via Telementry

## Description

Verify that the RHMI alerts are exposed via Telementry to cloud.redhat.com and that no RHMI alerts are firing.

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

5. Check after couple of hours (or next day) that no unexpected alert notifications were received
