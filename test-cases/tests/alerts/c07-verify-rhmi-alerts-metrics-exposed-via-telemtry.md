---
estimate: 15m
---

# C07 - Verify RHMI Alerts Metrics Exposed Via Telemtry

## Prerequisites

1. Login to the cluster manager console on https://qaprodauth.cloud.redhat.com/beta/openshift/

   - select the test cluster
   - select the Monitoring tab
   - expand Alerts firing from the menu
   - the DeadMansSwitch should be seen.

Note: If you follow the DeadMansSwitch link, at the moment you won't see the alert in OpenShift console, that will be added with <https://issues.redhat.com/browse/INTLY-4874> and <https://issues.redhat.com/browse/INTLY-6596>.
