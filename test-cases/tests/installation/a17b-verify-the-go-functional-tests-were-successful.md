---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - osd-private-post-upgrade
estimate: 15m
tags:
  - per-build
---

# A17B - Verify the Go functional tests were successful

Acceptance Criteria:

1. All tests need to report as PASSED

## Steps

1. Select the Jenkins job used for installation of the QE release testing cluster from

https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow

If there is not such build (typically if working with post upgrade cluster) then trigger the build

- provide "ocmAccessToken"
- provide display name of the cluster to "clusterName"
- tick multiAZ checkbox if cluster is deployed on multiple Availability Zones
- make sure only "runFunctionalTests" is ticked for Pipeline steps

2. Check the functional tests were run as part of the pipeline.
3. Check if there were any failed or skipped tests (flaky ones).
4. If so, then investigate failed tests, and re-run or verify manually any skipped tests.
5. If not, then the this test is passed.
