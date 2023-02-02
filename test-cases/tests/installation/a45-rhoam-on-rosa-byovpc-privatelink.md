---
products:
  - name: rhoam
    environments:
      - external
    targets:
      - 1.31.0
estimate: 3h
---

# A45 - RHOAM on ROSA BYOVPC private-link

## Prerequisites

- AWS account with ROSA enabled - the QE AWS account used for ROSA nightly pipeline can be used
- IAM user with console access to the AWS account
- OCM access to where cluster was created - OCM token from parent epic can be used

## Description

Verify RHOAM installation on ROSA + BYOVPC + multiAZ + --private-link works as expected.

## Steps

1. Create VPC

Do this via [AWS console](https://aws.amazon.com/console/). Log in, navigate to "CloudFormation" and follow the wizard to create a stack from [template](https://github.com/integr8ly/delorean/blob/master/templates/ocm/vpc-private-link-fw-us-east-1.yaml). Just click "Next" a few times, and on the last page tick the checkbox to allow for creating resources. Click "Submit" and wait for the stack to be created. The `us-east-1` region is hard-coded there so change that if using a different region. This one is recommended because not each region supports private-link and not each supports multiAZ and it is known that this one supports both.

2. Provision ROSA cluster via Delorean

Clone [delorean](https://github.com/integr8ly/delorean). Log into OCM. Log into AWS

`make ocm/rosa/cluster/create STS=true AWS_REGION=us-east-1 PRIVATE_LINK=true MULTI_AZ=true COMPUTE_NODES=6 SUBNET_IDS=subnet-04464f7727d1d80a3,subnet-01b7b485cf73f4bbf,subnet-068160018c71c4c14 CLUSTER_NAME=a45 MACHINE_CIDR=10.0.0.0/16 BYOVPC=true`

Use the three "Private" subnet IDs created as part of VPC creation.

Wait for the cluster to be ready and healthy (change the cluster ID as needed):

`ocm get subs --parameter search="cluster_id = '21jv00i5dgndlg00ps415f7mu2tvpen3'" | jq -r '.items[0].metrics[0].health_state'`

3. Install RHOAM addon

`rosa install addon managed-api-service --cluster a45 --addon-resource-required true --rosa-cli-required true --billing-model standard --region us-east-1`

This might fail with "Failed to verify operator role for cluster". Wait a minute and try again.

4. Patch the useClusterStorage

Connect to the bastion instance via "Session Manager" in AWS console. Click on "bastion" EC2 and click "Connect" to do that.

```
$ cd /home/ssm-user
$ curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/linux/oc.tar.gz
$ tar -xvf oc.tar.gz
$ sudo mv oc /usr/local/bin/
$ oc patch rhmi rhoam -n redhat-rhoam-operator --type=merge -p '{"spec":{"useClusterStorage": "false" }}'
```

5. Wait for RHOAM to be installed successfully

The `toVersion` should disappear from RHMI CR.

6. Run the test suite (the steps below should work from bastion)

Change the [Principal](https://github.com/integr8ly/delorean/blob/1855275dae30b8beaead789e01cb18ab7df46579/scripts/rosa/rosa.sh#L181) to `"Principal": "*"` to allow assuming the role for any user/service/session. Then execute from local terminal:
`make ocm/sts/sts-cluster-prerequisites CLUSTER_NAME=a45 ROLE_NAME=rhoam_role_jenkins FUNCTIONAL_TEST_ROLE_NAME=functional_test_role_jenkins`

(Optional) Setup testing-idp. This has to be done from bastion.

- `sudo yum install git jq docker`
- [download](https://github.com/openshift-online/ocm-cli/releases) and install ocm (similar to `oc` install above)
- log into OCM via ocm cli
- clone integreatly-operator repository
- `PASSWORD=Password1 DEDICATED_ADMIN_PASSWORD=Password1 ./scripts/setup-sso-idp`
- `sudo systemctl start docker`
- `cd /tmp`
- `mkdir test-results-01`
- update openshift host, password, aws account id, and execute the command below
- `sudo docker run -it -e OPENSHIFT_HOST='https://api.a45.ao06.s1.devshift.org:6443' -e OPENSHIFT_PASSWORD='<redacted>' -e MULTIAZ=true -e DESTRUCTIVE=false -e NUMBER_OF_TENANTS='2' -e TENANTS_CREATION_TIMEOUT='3' -e WATCH_NAMESPACE='redhat-rhoam-operator' -e BYPASS_STORAGE_TYPE_CHECK=true -e ROLE_ARN='arn:aws:iam::<aws-account-id>:role/functional_test_role_jenkins' -e TOKEN_PATH='/var/run/secrets/openshift/serviceaccount/token' -e LOCAL=false -e INSTALLATION_TYPE='managed-api' -e RegExpFilter='' -e OUTPUT_DIR=test-results-01 -v "$(pwd)/test-results-01:/test-results-01:Z" quay.io/integreatly/integreatly-operator-test-external:latest`

7. Analyze test results

It is known that H24 (self-managed apicast) might fail on

```
Performing OpenShift HTTP client setup with URL https://console-openshift-console.apps.trepel.4mq2.s1.devshift.org/auth/login
Expected status 200 but got 504
```

F01, F03, F04, A25 might fail on

```
arn:aws:sts::342316834583:assumed-role/SSMRole-multiaz-us-east-1/i-08acef83474d91295 is not authorized to perform: sts:AssumeRole on resource: arn:aws:iam::342316834583:role/functional_test_role_jenkins
```

You probably forgot to change the `Principal` to `"Principal": "*"` (see above). Alternatively you can only add just the required role session to [Principal.AWS](https://github.com/integr8ly/delorean/blob/1855275dae30b8beaead789e01cb18ab7df46579/scripts/rosa/rosa.sh#L181) so that the `Principal.AWS` should look similar to:

```
          "AWS": [
              "arn:aws:iam::$(get_account_id):user/osdCcsAdmin",
              "arn:aws:sts::342316834583:assumed-role/SSMRole-multiaz-us-east-1/i-08acef83474d91295"
          ],
```

See [here](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_principal.html) details.
Run the `ocm/sts/sts-cluster-prerequisites` make target again (see above). It re-creates the policy and the role. Execute the test suite as before, just use e.g. `-e RegExpFilter='F0.*'`
