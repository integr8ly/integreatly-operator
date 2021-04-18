---
components:
  - product-3scale
  - product-sso
estimate: 1h
products:
  - name: rhoam
    environments:
      - osd-fresh-install
tags:
  - per-release
---

# A30 - Validate installation of RHOAM addon and integration with LDAP IDP

## Description

We want to validate that customer is able to install RHOAM via OCM UI and can use LDAP server as a RHOAM IDP.

## Prerequisites

- access to [AWS secrets file in 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) (follow the guide in the [README](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/README.md) to unlock the vault with git-crypt key)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed locally
- login to [OCM UI (staging environment)](https://qaprodauth.cloud.redhat.com/beta/openshift/)
- access to the [spreadsheet with shared AWS credentials](https://docs.google.com/spreadsheets/d/1P57LhhhvhJOT5y7Y49HlL-7BRcMel7qWWJwAw3JCGMs)

## Steps

**Verify that OSD cluster can be created with provided credentials**

1. Go to the [spreadsheet with shared AWS credentials](https://docs.google.com/spreadsheets/d/1P57LhhhvhJOT5y7Y49HlL-7BRcMel7qWWJwAw3JCGMs) and select "AWS accounts" sheet
2. Look for AWS account ID that is free (doesn't have anything specified in 'Note'). If no account is free, you can use account that is used by nightly pipelines (but don't forget to clean it up for night)
3. Open the [AWS secrets file from 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) locally and look for the AWS credentials for the selected AWS account (aws account id, access key ID and secret access key)
4. Go to [OCM UI (staging environment)](https://qaprodauth.cloud.redhat.com/beta/openshift/) and log in
5. Click on `Create cluster` and select `OpenShift Dedicated`
6. Select AWS and click on `Customer cloud subscription` (and close the pop up notification)
7. Insert AWS account ID, access key ID and secret access key from the SECRETS.md file (from the step above)
8. Also fill in the following parameters:

```
Cluster name: test-ldap-idp
Availability: Multizone
Worker node count (per zone): 3
Networking: Advanced

Machine CIDR: 10.11.128.0/24
Service CIDR: 10.11.0.0/18
Pod CIDR: 10.11.64.0/18
Host prefix: /26
```

9. Click on `Create cluster` (cluster creation takes ~40 minutes)

**Verify RHOAM installation via addon**

1. Once the cluster is created, you can install the RHOAM addon
2. Select your cluster -> `Add-ons` and click on `Install`
3. Fill in the following parameters and click on `Install`

```
CIDR range: "10.1.0.0/26" (note this down to use it later for another verification step)
Notification email: "<your-username>+ID1@redhat.com <your-username>+ID2@redhat.com"
```

4. You should now login to your cluster via `oc` and patch RHMI CR to select the cloud-storage-type of installation:

```bash
# Copy your cluster's name from OCM UI ("test-ldap-idp" by default) and assign it to the env var CLUSTER_NAME
CLUSTER_NAME=<your-cluster-name>
# Get cluster's CID
CID=$(ocm get clusters --parameter search="display_name like '$CLUSTER_NAME'" | jq -r '.items[0].id')
# Get your cluster API URL and kubeadmin password
API_URL=$(ocm get cluster $CID | jq -r .api.url)
KUBEADMIN_PASSWORD=$(ocm get cluster $CID/credentials | jq -r .admin.password)
# Log in via oc
oc login $API_URL -u kubeadmin -p $KUBEADMIN_PASSWORD --insecure-skip-tls-verify=true
# Patch RHMI CR
oc patch rhmi rhoam -n redhat-rhoam-operator --type=merge -p '{"spec":{"useClusterStorage": "false" }}'
```

5. Now the installation of RHOAM should be in progress. You can watch the status of the installation with this command:

```
watch "oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage"
```

6. Once the status is "completed", the installation is finished and you can go to another step

> Due to a known issue the OCM UI can display `Installed` despite installation still being in progress

**Verify custom RHOAM CIDR range was applied correctly**

1. Export following variables (it is required to be logged into the OpenShift cluster as kubeadmin)

```bash
export AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o jsonpath={.data.aws_access_key_id} | base64 --decode) \
AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o jsonpath={.data.aws_secret_access_key} | base64 --decode) \
AWS_REGION=$(oc get infrastructure cluster -o jsonpath='{.status.platformStatus.aws.region}')
```

2. Run following AWS command

```bash
aws ec2 describe-vpcs --filters "Name=tag-key,Values=integreatly.org/clusterID" --region $AWS_REGION | jq -r '.Vpcs[0].CidrBlockAssociationSet[0].CidrBlock'
```

> Verify that the CIDR block you get on the output matches with the one that was specified in the installation form (via OCM UI) ^

**Note**

The scenario with **default CIDR range**, when user doesn't specify any CIDR, so it is automatically created by CRO, is covered by [the installation pipeline](https://github.com/integr8ly/delorean/blob/0cd8e05a49540c0c505c3c291629dd737d7cc818/scripts/ocm/ocm.sh#L144) - it doesn't provide any addon params, so CIDR block has to be created by CRO. If the pipeline finishes successfully, it means that the CIDR block was correct.

**Verify that LDAP IDP can be configured**

1. Go to [AWS secrets file in 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) again and search for "ldap-server-tester AWS UI credentials"
2. Access the AWS EC2 console with those credentials

3. On the AWS EC2 console, right click on the LDAP Server EC2 instance and select start instance.

   > Instance should change to running state

4. Once the instance is in a running state verify if the LDAP service is up and running.

   ```bash
   // you need to change the <EC2 instance IP> to the EC2 instance IP

   curl -vv "ldap://<EC2 instance IP>:389/dc=ec2-3-133-150-27,dc=us-east-2,dc=compute,dc=amazonaws,dc=com?uid?sub?(uid=rhoam)"

   DN: uid=rhoam,ou=people,dc=ec2-3-133-150-27,dc=us-east-2,dc=compute,dc=amazonaws,dc=com
    uid: rhoam
   ```

Create the integration with RHOAM via IDP.

5. Back in the OCM console, go to your cluster's details.

6. Click on the `Access Control` tab.

7. Press the `Add identity provider` button.

8. A form will pop up in a modal with three sections:

   4.1 Step 1: Select identity providers type

   - Select the LDAP option in the `Indentity provider` field

     4.2 Step 2: Enter provider type information

   - Leave the fields `Name` and `Mapping method` as it is

   - In `LDAP URL` add the URL we want to use for searching users in the LDAP server

   ```bash
   // you need to change the <EC2 instance IP> to the EC2 instance IP

   ldap://<EC2 instance IP>:389/dc=ec2-3-133-150-27,dc=us-east-2,dc=compute,dc=amazonaws,dc=com?uid?sub
   ```

   - Leave the fields Bind DN and Bind password empty

     4.3 Attributes

   - In ID field add `dn`

   - In `Preferred username` enter `uid`

   - In `Name` enter `cn`

   - Click on `Show Advanced Options` and mark the `Insecure` checkbox, so we don't need to configure certs

   - Press the `Confirm` button and wait for the changes to be reconciled

Verify the integration with the LDAP server for an admin user

1. Go to the cluster login page and check if there is an LDAP option for authentication in the list, if so click on this option.

2. Enter `rhoam-customer-admin` for _username_ and `Password1` for _password_ and click the log in button.

   > You should be redirected to the cluster console main page

3. In the terminal, log in on the cluster as kubeadmin.

   ```bash
   oc login --token=<token> --server=https://api.<cluster_domain>:6443
   ```

4. Promote `rhoam-customer-admin` to the `dedicated-admin` group by running the command below:

   ```bash
   oc adm groups add-users dedicated-admins rhoam-customer-admin && oc adm groups remove-users rhmi-developers rhoam-customer-admin
   ```

5. Get the 3scale admin URL that matches with `https://3scale-admin.apps.<cluster-id>.devshift.org`.

   ```bash
   oc get route -n redhat-rhoam-3scale
   NAME                         HOST/PORT                                                          PATH   SERVICES             PORT      TERMINATION     WILDCARD
   backend                      backend-3scale.apps.<cluster-id>.devshift.org                         backend-listener     http      edge/Allow      None
   zync-3scale-api-*        api-3scale-apicast-staging.apps.<cluster-id>.devshift.org             apicast-staging      gateway   edge/Redirect   None
   zync-3scale-api-*        api-3scale-apicast-production.apps.<cluster-id>.devshift.org          apicast-production   gateway   edge/Redirect   None
   zync-3scale-master-*     master.apps.<cluster-id>.devshift.org                                 system-master        http      edge/Redirect   None
   zync-3scale-provider-*   3scale.apps.<cluster-id>.devshift.org                                 system-developer     http      edge/Redirect   None
   zync-3scale-provider-*   3scale-admin.apps.<cluster-id>.devshift.org                           system-provider      http      edge/Redirect   None

   ```

6. Open an incognito window and paste the url in.

   > You should be redirected to the 3scale login page

7. Click on the `Authenticate through Red Hat Single Sign-On` link.

   > You should be redirected to Red Hat single sign on page

8. Click on the LDAP option.

9. Enter `rhoam-customer-admin` for _username_ and `Password1` for _password_ and click the log in button.

   > You should be redirected to the 3scale main page.

Verify the integration with the LDAP server for a regular user

1. Go to the cluster login page and check if there is an LDAP option for authentication in the list, if so click on this option.

2. Enter `rhoam-test-user` for _username_ and _Password1_ for _password_ and click the log in button

   > You should be redirected to the cluster console main page

3. In the terminal, log in on the cluster as `kubeadmin`

   ```bash
   oc login --token=<token> --server=https://api.<cluster_domain>:6443
   ```

4. Get the 3scale admin URL that matches with `https://3scale-admin.apps.<cluster-id>.devshift.org`

   ```bash
   oc get route -n redhat-rhoam-3scale
   NAME                         HOST/PORT                                                          PATH   SERVICES             PORT      TERMINATION     WILDCARD
   backend                      backend-3scale.apps.<cluster-id>.devshift.org                         backend-listener     http      edge/Allow      None
   zync-3scale-api-*        api-3scale-apicast-staging.apps.<cluster-id>.devshift.org             apicast-staging      gateway   edge/Redirect   None
   zync-3scale-api-*        api-3scale-apicast-production.apps.<cluster-id>.devshift.org          apicast-production   gateway   edge/Redirect   None
   zync-3scale-master-*     master.apps.<cluster-id>.devshift.org                                 system-master        http      edge/Redirect   None
   zync-3scale-provider-*   3scale.apps.<cluster-id>.devshift.org                                 system-developer     http      edge/Redirect   None
   zync-3scale-provider-*   3scale-admin.apps.<cluster-id>.devshift.org                           system-provider      http      edge/Redirect   None

   ```

5. Open an incognito window and paste the url in.

   > You should be redirected to the 3scale login page

6. Click on the `Authenticate through Red Hat Single Sign-On` link

   > You should be redirected to Red Hat single sign on page

7. Click on the LDAP option.

8. Enter `rhoam-test-user` for _username_ and `Password1` for _password_ and click the log in button.

   > You should be redirected to the 3scale main page.

**Verify RHOAM uninstall**

1. Go to OCM UI, select your cluster, go to Add-ons tab
2. Click on the three dots menu and select "Uninstall add-on"
   > Verify that RHOAM addon updates the status to "Uninstalling"
3. Go to OpenShift console of your cluster
   > In a while you should see RHOAM namespaces disappearing
   > Uninstall of RHOAM should finish in ~20 minutes (all `redhat-rhoam-*` namespaces should be gone)
4. If there's still `redhat-rhoam-operator` namespace present after 20 minutes, follow the next steps, otherwise skip to the last step
5. Go to `redhat-rhoam-operator` namespace and select Operators -> Installed operators
   > Verify that there's no operator subscription present
6. Go to `redhat-rhoam-operator` namespace and search for "RHMI"
   > Verify that there's nothing in the results
7. If there's no subscription or RHMI CR present in `redhat-rhoam-operator` namespace, you have to wait for `hive` to finish the uninstall (sometimes it could take more than 10 minutes)
8. Go to OCM UI, select your cluster, go to Add-ons tab and verify that RHOAM addon is uninstalled (you should see a button "Install" again)

**Verify RHOAM secrets are recreated on reinstall and RHOAM can be uninstalled during "preflight checks" phase**

1. From OCM UI, trigger RHOAM install again
2. Go to OpenShift console and select `redhat-rhoam-operator` namespace
3. Go to Workloads -> Secrets
   > Verify that `redhat-rhoam-deadmanssnitch`, `redhat-rhoam-pagerduty` and `redhat-rhoam-smtp` are present
4. Go to OCM UI, select your cluster
5. Note your cluster's ID from the address bar (https://qaprodauth.cloud.redhat.com/beta/openshift/details/<cluster_id>#overview)
6. Delete the addon via ocm CLI

```bash
ocm delete /api/clusters_mgmt/v1/clusters/<CLUSTER-ID>/addons/managed-api-service
```

7. Go back to OpenShift console and verify RHMI CR was removed as well as and `redhat-rhoam-operator` namespace

**As the test is finished lets stop the EC2 instance.**

1. Access the AWS EC2 console, you can find the credentials in https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md

2. On the AWS EC2 console, right click on the LDAP Server EC2 instance and select stop instance.

   > Instance should change to stopped state.
