---
products:
  - name: rhoam
    environments:
      - external
estimate: 2h
tags:
  - per-release
---

# A37 - RHOAM on ROSA

## Description

Verify RHOAM installation on ROSA works as expected.

## Steps

1. Login to [OCM UI (staging environment)](https://qaprodauth.console.redhat.com/beta/openshift/)
2. Get the [OCM API Token](https://qaprodauth.console.redhat.com/beta/openshift/token)
3. Trigger the pipeline and specify your ocmAccessToken in the pipeline parameters.
4. Validate pipeline results

> [ROSA pipeline](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-rosa/)
