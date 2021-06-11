---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - rhpds
      - osd-private-post-upgrade
estimate: 15m
tags:
  - per-build
---

# A17A - Verify the Go functional tests were successful

Acceptance Criteria:

1. All tests need to report as PASSED

## Steps

1. Select the Jenkins job used for installation of the QE release testing cluster from

https://master-jenkins-csb-intly.apps.ocp4.prod.psi.redhat.com/job/Integreatly/job/rhmi-install-addon-flow

2. Check the functional tests were run as part of the pipeline.
3. Check if there were any failed or skipped tests (flaky ones).
4. If so, then investigate failed tests, and re-run or verify manually any skipped tests.
5. If not, then the this test is passed.
