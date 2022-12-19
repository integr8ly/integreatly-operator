---
components:
  - product-3scale
  - product-sso
estimate: 1h
products:
  - name: rhoam
    environments:
      - external
tags:
  - manual-selection
---

# A31 - Verify installation on all OSD versions

## Description

We want to validate that RHOAM can be installed via Addon Flow on all currently available OSD versions.

## Prerequisites

- Access to [AWS secrets file in 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) (follow the guide in the [README](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/README.md) to unlock the vault with git-crypt key)
- Login to [OCM UI (staging environment)](https://qaprodauth.console.redhat.com/beta/openshift/)
- Access to the [spreadsheet with shared AWS credentials](https://docs.google.com/spreadsheets/d/1P57LhhhvhJOT5y7Y49HlL-7BRcMel7qWWJwAw3JCGMs)
- Access to [RHOAM Addon Flow pipeline](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow/)

## Steps

**Find OSD versions to be validated**

1. Login to [OCM UI (staging environment)](https://qaprodauth.console.redhat.com/beta/openshift/)
2. Get the [OCM API Token](https://qaprodauth.console.redhat.com/beta/openshift/token)
3. Use the command displayed on the screen after the step above to log in to ocm cli
4. `ocm cluster versions`
5. Repeat the testing for latest micro (patch) of current minor version and minor -1 and minor -2
   > e.g. if `4.4.20`, `4.5.16`, `4.5.17`, and `4.6.0` are available, test with `4.4.20`, `4.5.17` and `4.6.0`

**Trigger the pipeline and analyse the test results**

1. Go to the [RHOAM Addon Flow pipeline](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow/) and select "Build with Parameters"
2. Look for AWS account ID that is free (it doesn't have anything specified in 'Note'). If no account is free, you can use account that is used by nightly pipelines (but don't forget to clean it up for night)
3. Open the [AWS secrets file from 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) locally and look for the AWS credentials for the selected AWS account (aws account id, access key ID and secret access key)
4. Make sure to fill in `openshiftVersion` properly
5. Make sure to use your own ocmAccessToken so you have access to the cluster provisioned by the pipeline.
6. Most of the parameters are self-explanatory, many can be left as they are or blank

```
clusterComputeNodesCount: 6
multiAZ: true (tick the checkbox)
adminPassword: Password1
```

6. Wait till the pipeline finishes
7. Analyse the failed tests (ideally there are none)
8. Log in as customer-admin01 to the 3scale and User SSO
9. Log in as test-user01 to the 3scale and User SSO
