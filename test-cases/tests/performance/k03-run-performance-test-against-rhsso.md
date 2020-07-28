---
components:
  - product-sso
environments:
  - osd-post-upgrade
estimate: 2h
tags:
  - destructive
targets:
  - 2.8.0
---

# K03 - Run performance test against RHSSO

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Prerequisites

You should have an AWS IAM account in [QE-AWS account](https://068334777414.signin.aws.amazon.com/console)

## Steps

1. Get access to ec2 instance to run tests
   - Option 1 - Use existing ec2 instance:
     1. Contact Stephin Thomas for details on accessing this ec2 instance
   - Option 2 - Create new ec2 instance:
     1. Login into [QE-AWS account](https://068334777414.signin.aws.amazon.com/console) using your AWS IAM account credentials and select the region to **London**.
     2. Create a new **t2.small** ec2 instance from the sso-load-testing AMI. Set the Auto-assign public IP to Enable and select Network as **sso-load-test-vpc** and subnet as **sso-load-test-subnet**. Configure security group to SSH with "myIP".
2. SSH into the instance using the public IPv4 address and the centos user. `ssh -i <pemfilename>.pem centos@<public IP>`
3. `cd keycloak` and `git fetch --tags`
4. Checkout the Keycloak tag corresponding with RHSSO version on cluster
   - for example, Keycloak 10 = RHSSO 7.4
5. `cd testsuite/performance/`
6. Run `mvn clean install`
7. Replace the username, password and server properties of `tests/target/provisioned-system.properties` with your clusters RHSSO details.

   > keycloak.frontend.servers= `<rhsso-url>`
   >
   > keycloak.admin.user= `<admin-user>`
   >
   > keycloak.admin.password= `<admin-pwd>`

8. Run `mvn verify -Pgenerate-data -Ddataset=1r_10c_100u -DnumOfWorkers=10 -Djackson.version=2.10.1`
   - **NOTE:** `-Djackson.version=2.10.1` is a workaround for RHSSO 7.4 only. Please remove this param from the test case when we upgrade to RHSSO 7.5
9. Run

```
export WARMUP_PERIOD=900
export RAMPUP_PERIOD=120
export MEASUREMENT_PERIOD=240
export STRESS_TEST_UPS_FIRST=14
./stress-test.sh -DmaxMeanReponseTime=1000
```

10. Once tests have finished, results will be saved to `/home/centos/keycloak/testsuite/performance/tests/target/gatling/`. Copy the last two folders to your local machine to view the tests using something similar to `scp -r -i ../stephin.pem centos@ec2-3-8-28-1.eu-west-2.compute.amazonaws.com:/home/centos/keycloak/testsuite/performance/tests/target/gatling/oidcloginandlogoutsimulation-1580403162107 ./`

> Capture the results and compare them against [the previous run](https://docs.google.com/spreadsheets/d/1VGL87kaSKaz7ndjj1tNlRQiYDf2zn-lT1uHeOCPII3M/edit#gid=1845969669)
>
> Expecting similar results
