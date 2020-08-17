---
environments:
  - osd-fresh-install
tags:
  - manual-selection
---

# I16 - Verify partial ag maintenance and upgrades

## Description

Procedures for scheduling RHMI cluster maintenance windows and upgrades. This test case actually covers two JIRAs. The procedures in these JIRAs go together and should be tested at the same time.

- Merge Requests:
  - Maintenance windows: https://gitlab.cee.redhat.com/RedHatManagedIntegration-documentation/rhmi-docs/-/merge_requests/174
  - Upgrades: https://gitlab.cee.redhat.com/RedHatManagedIntegration-documentation/rhmi-docs/-/merge_requests/173
- JIRAs:
  - Maintenance windows: https://issues.redhat.com/browse/RHMIDOC-108
  - Upgrades: https://issues.redhat.com/browse/RHMIDOC-106
- Peer contacts: Joan Edwards and Ben Hardesty
- Preview doc links:
  - Maintenance windows: https://cee-jenkins.rhev-ci-vms.eng.rdu2.redhat.com/job/CCS/job/ccs-mr-preview/16062/artifact/docs/admin-guide/preview/index.html#scheduling-maintenance-windows_admin-guide
  - Upgrades: https://cee-jenkins.rhev-ci-vms.eng.rdu2.redhat.com/job/CCS/job/ccs-mr-preview/16125/artifact/docs/admin-guide/preview/index.html#scheduling-upgrades_admin-guide

## Steps

1. Navigate to the [preview doc for scheduling maintenance windows](https://cee-jenkins.rhev-ci-vms.eng.rdu2.redhat.com/job/CCS/job/ccs-mr-preview/16062/artifact/docs/admin-guide/preview/index.html#scheduling-maintenance-windows_admin-guide).

2. Test 3.1.2. Scheduling maintenance windows.

   > Prerequisites are correct.

   > Steps are clear.

   > Maintenance window schedule is correct after clicking "Save".

3. Navigate to the [preview doc for scheduling upgrades](https://cee-jenkins.rhev-ci-vms.eng.rdu2.redhat.com/job/CCS/job/ccs-mr-preview/16125/artifact/docs/admin-guide/preview/index.html#scheduling-upgrades_admin-guide).

4. Test 3.1.3. Scheduling upgrades.

   > Prerequisites are correct.

   > Steps are clear.

   > Maintenance window schedule is correct after clicking "Save".
