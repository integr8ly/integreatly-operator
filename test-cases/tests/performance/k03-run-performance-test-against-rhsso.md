---
components:
  - product-sso
environments:
  - osd-post-upgrade
estimate: 2h
tags:
  - destructive
targets:
  - 2.6.0
---

# K03 - Run performance test against RHSSO

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

Note: for details see various README files in performance directory in [keycloak](https://github.com/keycloak/keycloak/tree/master/testsuite/performance) repository.

## Prerequisites

In case you can't (or don't want to) use Stephin's ec2 instance (option1, see below), you will need to have an AWS IAM account in [QE-AWS account](https://068334777414.signin.aws.amazon.com/console).

### How to create Trust Store JKS file

In case you are receiving 404 errors or SSL handshake issues, you might need to use JKS Trust Store.

1. Export the certificate (PEM format) via browser by navigating to the keycloak URL and clicking on the lock next to the URL. This differs between browsers but you should be able to find 'Export' button eventually.
2. Download and run [Portecle](http://portecle.sourceforge.net/)
3. File -> New Keystore
4. Tools -> Import Trusted Certificate, use the file exported in step 1
5. File -> Save Keystore
6. To verify: `keytool -list -keystore yourNewlyCreatedTrustStore.jks`, you should see one entry there

## Steps

1. Get access to ec2 instance to run tests
   - Option 1 - Use existing ec2 instance:
     1. Clone and decrypt the [vault](https://gitlab.cee.redhat.com/integreatly-qe/vault) repo
     2. ssh -i <PATH-TO-VAULT-REPO>/keys/sso-load-test-ec2 centos@ec2-3-8-28-1.eu-west-2.compute.amazonaws.com
   - Option 2 - Create new ec2 instance:
     1. Login into [QE-AWS account](https://068334777414.signin.aws.amazon.com/console) using your AWS IAM account credentials and select the region to **London**.
     2. Create a new **t2.small** ec2 instance from the sso-load-testing AMI. Set the Auto-assign public IP to Enable and select Network as **sso-load-test-vpc** and subnet as **sso-load-test-subnet**. Configure security group to SSH with "myIP".
2. SSH into the instance using the public IPv4 address and the centos user. `ssh -i <pemfilename>.pem centos@<public IP>`
3. `cd keycloak` and `git fetch --tags`
4. Checkout the Keycloak tag corresponding with RHSSO version on cluster
   - for example, Keycloak 10 = RHSSO 7.4
5. `cd testsuite/performance/`
6. Run `mvn clean install`

   > you might need to delete/move the `tests/target/provisioned-system.properties` file for the command to pass. Once finished, put the file back.

7. Replace the username, password and server properties of `tests/target/provisioned-system.properties` with your clusters RHSSO details. Make sure `/auth` is appended at the end of RHSSO URL.

   > keycloak.frontend.servers= `<rhsso-url>/auth`
   >
   > keycloak.admin.user= `<admin-user>`
   >
   > keycloak.admin.password= `<admin-pwd>`

8. Run `mvn verify -Pgenerate-data -Ddataset=1r_10c_100u -Djackson.version=2.10.1`

   > this should create `realm_0` realm and 100 users in that realm. If both realm and users already exists, this can be skipped. It is idempotent, running it more times makes no harm though.

   - **NOTE:** `-Djackson.version=2.10.1` is a workaround for RHSSO 7.4 only. Please remove this param from the test case when we upgrade to RHSSO 7.5

   - **NOTE:** Setting the JKS truststore should not be required for this command. In case of 404 errors try to use 9.0.3 tag of keycloak repository, see [KEYCLOAK-15409](https://issues.redhat.com/browse/KEYCLOAK-15409).

9. Execute single performance test

   `mvn verify -Ptest -Ddataset=1r_10c_100u`

   > This is just to verify that everything is ok before running the actuall stress test. If the command passes ok you can proceed with the next step.

   > If experiencing 404 not found issues, double check the RHSSO URL, make sure the `/auth` is appended. It might also be required to set Java Trust Store using `-DtrustStore=PATH_TO/TrustStore.jks -DtrustStorePassword=<password-if-it-is-password-protected>`.

10. Run

```
export WARMUP_PERIOD=900
export RAMPUP_PERIOD=120
export MEASUREMENT_PERIOD=240
export STRESS_TEST_UPS_FIRST=14
./stress-test.sh -DmaxMeanReponseTime=1000
```

> Read [here](https://github.com/keycloak/keycloak/blob/master/testsuite/performance/README.stress-test.md#stress-test) for more details about the script.

> The script runs the single performance test in a loop (iteration) and typically it fails with an error (it might also finish after max. number of iterations has been run, in that case increase the number of iterations (STRESS_TEST_MAX_ITERATIONS env variable) until it fails):

```
INFO: Last iteration failed. Stopping the loop.
Maximal load with passing test: 16.000 users per second
```

11. Once tests have finished, results will be saved to `/home/centos/keycloak/testsuite/performance/tests/target/gatling/`. Copy the last two folders (reports of last two iterations) to your local machine. Use someting similar to `scp -r -i sso-load-test-ec2 centos@ec2-3-8-28-1.eu-west-2.compute.amazonaws.com:/home/centos/keycloak/testsuite/performance/tests/target/gatling/oidcloginandlogoutsimulation-<timestamp> ./`.

> Capture the results and compare them against [the previous run](https://docs.google.com/spreadsheets/d/1VGL87kaSKaz7ndjj1tNlRQiYDf2zn-lT1uHeOCPII3M/edit#gid=1845969669)
> It might be useful to search for existing JIRA tickets and download previous reports. Then compare the full reports.
> Expecting similar results
> Make sure to add any relevant information (keycloak repo tag, RHMI version, SSO version) to the JIRA ticket comments for later review
