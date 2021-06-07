---
products:
  - name: rhoam
    environments:
      - external
estimate: 3h
tags:
  - per-release
---

# A39 - installation - verify RHOAM on private cluster

## Steps

1. Create a clean OSD cluster as usual
2. Follow [this guide](https://docs.google.com/document/d/1BwjzezNFtE7gd2y6FY6v2W6KRXCn0jMZk58ilJ8zSa8/edit) to make it private
3. Use [addon-flow](https://master-jenkins-csb-intly.apps.ocp4.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow/) pipeline to run the automated tests against the cluster

- provide "ocmAccessToken"
- provide display_name of your private cluster to "clusterName"
- make sure only "installProduct", "setupIdp", and "runFunctionalTests" are ticked as "Pipeline steps"
