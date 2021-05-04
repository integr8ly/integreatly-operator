---
products:
  - name: rhoam
    targets:
      - 0.1.0
      - 1.2.0
      - 1.6.0
    environments:
      - osd-post-upgrade
tags:
  - destructive
---

# C17B - Verify RHOAM Upgrade alerts are present

## Description

Verify that RHOAM operator upgrade alerts are in place and firing

More info: https://issues.redhat.com/browse/INTLY-7551

## Prerequisites

- Bundles for the operator target version as well as an upgrade that replaces
  the target version release (to simulate the upgrade)

## Steps

1. Install RHOAM through OLM and wait for the installation to complete
2. Simulate an upcoming upgrade and let the operator approve it through the RHOAMConfig CR
3. Verify that the upgrade is completed successfully, and the RHOAM CR version is
   updated to the new release version
4. Verify the `RHOAMUpgradeExpectedDurationExceeded` alert is present:
   1. Open OpenShift console in your browser and login as admin
   2. Login as admin
   3. Find route for Prometheus in `openshift-monitoring` namespace
   4. Open its URL
   5. Go to the Alerts tab and look for `RHOAMUpgradeExpectedDurationExceeded`
   6. Click on the `expr`
   7. Select `Graph` and execute the query
   8. Verify that the value was `1` during the operator upgrade but was set back to empty when the upgrade completed
