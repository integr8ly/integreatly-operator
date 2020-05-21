---
estimate: 2h
---

# K03 - Run performance test against RHSSO

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Prerequisites

You should have an AWS IAM account in [QE-AWS account](https://068334777414.signin.aws.amazon.com/console)

## Steps

1. Login into [QE-AWS account](https://068334777414.signin.aws.amazon.com/console) using your AWS IAM account credentials and select the region to **London**.
2. Create a new **t2.small** ec2 instance from the sso-load-testing AMI. Set the Auto-assign public IP to Enable and select Network as **sso-load-test-vpc** and subnet as **sso-load-test-subnet**. Configure security group to SSH with "myIP".
3. SSH into the instance using the public IPv4 address and the centos user. `ssh -i <pemfilename>.pem centos@<public IP>`
4. `cd keycloak` and `git pull`
5. `cd testsuite/performance/`
6. Replace the username, password and server properties of `tests/target/provisioned-system.properties` with your clusters RHSSO details.

   > keycloak.frontend.servers= `<rhsso-url>`
   >
   > keycloak.admin.user= `<admin-user>`
   >
   > keycloak.admin.password= `<admin-pwd>`

7. Run `mvn clean install`
8. Run `mvn verify -Pgenerate-data -Ddataset=1r_10c_100u -DnumOfWorkers=10`
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
