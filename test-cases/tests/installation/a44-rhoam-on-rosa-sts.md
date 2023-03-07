---
products:
  - name: rhoam
    environments:
      - external
    targets:
      - 1.27.0
      - 1.30.0
      - 1.31.0
estimate: 2h
tags:
  - automated
---

# A44 - RHOAM on ROSA STS

## Description

Verify RHOAM installation on ROSA STS cluster. Related [documentation](https://docs.google.com/document/d/17_XTqdN0d7lU-SNHR3NArED6Jcl8Flm6ITb8BhNOBuU/edit#heading=h.xd9fvhhms75y)

## Prerequisites

- AWS account with ROSA enabled - the QE AWS account used for ROSA nightly pipeline can be used
- OCM access to where cluster was created - OCM token from parent epic can be used

## Steps

1. create ROSA STS cluster

`CLUSTER_NAME=rc1-sts AWS_REGION=us-west-1 ENABLE_AUTOSCALING=false STS_ENABLED=true ROLE_NAME=rc1sts_role_name FUNCTIONAL_TEST_ROLE_NAME=rc1sts_functional_test_role_name make ocm/rosa/cluster/create`
Use [https://github.com/integr8ly/delorean/pull/284] if not merged yet. It uses ROSA CLI under the hood, make sure you are logged into AWS account with ROSA enabled (if using QE AWS account for ROSA use `rosa-admin` IAM user). You can check via `rosa whoami` command.

2. Create IAM user for s3 bucket creation

See the doc above, there's a section on how to do it. In QE AWS account with ROSA enabled is used, the s3-3scale user is already created, see [vault](https://gitlab.cee.redhat.com/integreatly-qe/vault) for credentials. To be used later when installing the RHOAM addon.

3. Install RHOAM addon via rosa cli

`rosa install addon --cluster rc1-sts managed-api-service --cidr-range 10.1.0.0/26 --addon-managed-api-service 1 --addon-resource-required true --s3-access-key-id <redacted> --s3-secret-access-key <redacted> --billing-model standard`

For s3 access key ID and secret access key use the IAM user created in the previous step. Watch `sts-credentials` secret creation in CRO operator ns, it blocks RHOAM installation and it should be created by Hive. It can take up to 2 hours.

4. Trigger automated test suite once RHOAM installation succeeds

AWS tests would fail due to `failed to get STS credentials: ROLE_ARN key should not be empty`. One has to prepare role and policy as described in https://github.com/integr8ly/delorean/blob/master/scripts/sts/sts.sh#L224-L256. Note that the script is renamed to `rosa.sh` in the PR mentioned in step 1.

These two variables have to be added to test container:

- TOKEN_PATH=/var/run/secrets/openshift/serviceaccount/token
- ROLE_ARN=arn:aws:iam::<your-aws-account>:role/rc1sts_functional_test_role_name

Thus it is better to run the tests locally:

- navigate to where the [Delorean](https://github.com/integr8ly/delorean) repository is cloned (you can use the PR from step 1.
- `make build/cli`
- create a `test-config.yaml` file with following content

```
---

tests:
- name: integreatly-operator-test
  image: quay.io/integreatly/integreatly-operator-test-harness:master
  timeout: 7200
  envVars:
  - name: DESTRUCTIVE
    value: 'false'
  - name: MULTIAZ
    value: 'false'
  - name: WATCH_NAMESPACE
    value: redhat-rhoam-operator
  - name: BYPASS_STORAGE_TYPE_CHECK
    value: 'true'
  - name: LOCAL
    value: 'false'
  - name: INSTALLATION_TYPE
    value: managed-api
  - name: TOKEN_PATH
    value: '/var/run/secrets/openshift/serviceaccount/token'
  - name: ROLE_ARN
    value: arn:aws:iam::<your-aws-account>:role/rc1sts_functional_test_role_name
```

- `KUBECONFIG=<path/to/kubeconfig/file> ./delorean pipeline product-tests --test-config test-config.yaml --output test-results --namespace test-functional | tee testOutput.txt`

5. Analyse the failures if any
