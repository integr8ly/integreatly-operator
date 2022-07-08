---
products:
  - name: rhoam
    environments:
      - osd-private-post-upgrade
estimate: 3h
tags:
  - per-release
---

# A39 - installation - verify RHOAM on private cluster

## Steps

1. Create a clean OSD cluster as usual

- can be done either manually as described in [A30](./a30-validate-installation-of-rhoam-addon-and-integration-with-ld.md) or via [addon-flow](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow) pipeline
- it is enough for the cluster to be single-AZ

2. Follow [this guide](https://docs.google.com/document/d/1BwjzezNFtE7gd2y6FY6v2W6KRXCn0jMZk58ilJ8zSa8/edit) to make it private
3. Install RHOAM on the cluster

- manually via OCM CLI, see [A30](./a30-validate-installation-of-rhoam-addon-and-integration-with-ld.md)

4. Run the RHOAM functional test suite locally

- navigate to where the [Delorean](https://github.com/integr8ly/delorean) repository is cloned
- `make build/cli`
- create a `test-config.yaml` file with following content

```
---

tests:
- name: integreatly-operator-test
  image: quay.io/integreatly/integreatly-operator-test-harness:rhoam-latest-staging
  timeout: 7200
  envVars:
  - name: DESTRUCTIVE
    value: 'false'
  - name: MULTIAZ
    value: 'false'
  - name: WATCH_NAMESPACE
    value: redhat-rhoam-operator
```

- `KUBECONFIG=<path/to/kubeconfig/file> ./delorean pipeline product-tests --test-config test-config.yaml --output test-results --namespace test-functional | tee testOutput.txt`

5. Attach the test results to the ticket, analyze failures if any
