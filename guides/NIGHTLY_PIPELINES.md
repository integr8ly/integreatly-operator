# Nightly Pipelines

This guide outlines how to find, diagnose, take action and report on the RHMI 2.x nightly pipelines. 

It assumes no prior exposure to the integreatly-operator or the pipelines.



## Resources

- Documentation

  - [Detailed overview of RHMI 2.x nightly pipelines](https://docs.google.com/document/d/1eYtezxJHgOWENs_KRL27gp6AQmggCLUMVkIrSaNIcEQ/edit)

- Pipelines

  - [Master Installation](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Nightly/job/rhmi-install-master/) 
  - [Addon Installation](https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/Nightly/job/rhmi-install-addon-flow/)

- Pipeline configuration

  - [Pipeline configuration repo](https://gitlab.cee.redhat.com/integreatly-qe/ci-cd) 

> You will need to be on the Red Hat vpn to access Gitlab.
> Login to Jenkins and Gitlab is with your kerberos credentials. 




## Flaky tests

Pipelines runs may fail because of test(s) that pass/ fail inconsistently, ie a test produces a different result when re-run even though nothing related to the test has changed. These tests are regarded as `flaky`. 

Investigate the test failure(s) you are seeing by first verifying whether the test is a known flaky test.

Tests discovered to be flaky, and the process of handling flaky tests are detailed in this epic:

- https://issues.redhat.com/browse/INTLY-7197.

To verify if a test might be flaky run the test suite again locally against the cluster and verify if the results change:

```
oc login --server <api-url> -u <user> -p <password> 
git clone https://github.com/integr8ly/integreatly-operator.git
cd integreatly-operator
make test/functional
```



## Common issues

### Installation failures

TODO


### E2E test failures

TODO


## General troubleshooting

### RHMI CR

Check the RHMI CR for any errors:
- oc: `oc get rhmi rhmi -n redhat-rhmi-operator -o yaml`
- Openshift console: 


### RHMI Operator

Check the RHMI Operator logs for any errors:
- oc ``


### RHMI Monitoring and Alerts

TODO


### Product deployments

TODO


## Pipeline Results

Results from the previous nights pipeline runs are logged in the below locations:

- [Master Installation Results](https://docs.google.com/spreadsheets/d/1RY3Y7oKcBOyJRrFrXqnU7xI6M2UwkwU0_1DUY4cYAh0/edit#gid=1130966231)
- [Addon Installation Results](https://docs.google.com/spreadsheets/d/1RY3Y7oKcBOyJRrFrXqnU7xI6M2UwkwU0_1DUY4cYAh0/edit#gid=1130966231)

Results are highlighted for quick reference:

- Green: OK
- Red: Failed
- Yellow: External Failure 
- Orange: Re-run successful
- Grey: Not Available/ Tests did not execute



## Further help

If the above does not help resolve your issues you can reach out to the core-operator directly:

- Google chat channel: [integreatly-core-operator](https://chat.google.com/room/AAAAP43TtLA) (add `@hey core` to the start of your message)