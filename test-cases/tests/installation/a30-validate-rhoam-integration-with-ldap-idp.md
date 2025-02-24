---
components:
  - product-3scale
  - product-sso
estimate: 1h
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.27.0
      - 1.30.0
      - 1.33.0
      - 1.36.0
      - 1.39.0
      - 1.42.0
---

# A30 - Validate RHOAM integration with LDAP IDP

## Description

We want to validate that customer can use LDAP server as a RHOAM IDP.

## Prerequisites

- access to [AWS secrets file in 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) (follow the guide in the [README](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/README.md) to unlock the vault with git-crypt key)
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed locally
- login to [OCM UI (staging environment)](https://qaprodauth.console.redhat.com/beta/openshift/)

## Steps

**Verify that LDAP IDP can be configured**

1. Go to [AWS secrets file in 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) again and search for "RHOAM LDAP Instance" You should find "Login page" there and AWS credentials for AWS CLI
2. Access the AWS EC2 console with those credentials

   - you can either use the IAM user if you have one and navigate to "Login page" in your browser
   - or you can use AWS CLI

3. On the AWS EC2 console, right click on the LDAP Server EC2 instance and select start instance.

   > Instance should change to running state

   If having only AWS CLI access you can do the following:

   3.1 Retrieve the AWS Access Key ID and AWS Secret Access Key from [AWS secrets file in 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) for the mgdapi-5 AWS account.

   3.2 Configure AWS credentials

   ```bash
   aws configure
   AWS Access Key ID [****************RTNY]: <Access Key ID>
   AWS Secret Access Key [****************D6qv]: <Aws Secret Access Key>
   Default region name [us-east-2]: eu-west-1
   Default output format [None]: json
   ```

   3.3 Start the LDAP instance

   ```bash
   export AWS_PAGER=""
   EC2=`aws ec2 describe-instances --region eu-west-1 --filters "Name=tag:Name,Values=rhoam-ldap" --query 'Reservations[0].Instances[0].InstanceId' --output text`
   aws ec2 start-instances --region eu-west-1 --instance-ids $EC2
   aws ec2 wait instance-status-ok --instance-ids $EC2 --region eu-west-1

   # store the public IP for later use
   LDAP_PUBLIC_IP=$(aws ec2 describe-instances --region eu-west-1 --filters "Name=tag:Name,Values=rhoam-ldap" --query 'Reservations[0].Instances[0].PublicIpAddress' --output text)
   ```

4. Once the instance is in a running state verify if the LDAP service is up and running.

   ```bash
   curl -vv "ldap://${LDAP_PUBLIC_IP}/dc=ec2-172-31-37-63,dc=eu-west-1,dc=compute,dc=amazonaws,dc=com?uid?sub?(uid=rhoam-customer-admin)"

   DN: uid=rhoam-customer-admin,ou=people,dc=ec2-172-31-37-63,dc=eu-west-1,dc=compute,dc=amazonaws,dc=com
    uid: rhoam-customer-admin
   ```

   If the LDAP service is not running ssh to the EC2 instance and start it. Follow our [LDAP guide](https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/tree/master/qe-guides/rhoam-ldap-instance.md) to do so.

Create the integration with RHOAM via IDP.

5. Add LDAP as identity provider.
   5.1 Set LDAP_URL variable.

```bash
LDAP_URL=ldap://${LDAP_PUBLIC_IP}:389/dc=ec2-172-31-37-63,dc=eu-west-1,dc=compute,dc=amazonaws,dc=com?uid?sub
```

5.2 Retrieve cluster id

```bash
   CLUSTER_NAME=<your-cluster-name>
```

```bash
   CLUSTER_ID=$(ocm get clusters --parameter search="name like '$CLUSTER_NAME'" | jq -r '.items[0].id')
```

5.3 Add LDAP as identity provider.

```bash
ocm post https://api.stage.openshift.com/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers --body=<<EOF
{
   "type":"LDAPIdentityProvider",
   "name":"LDAP",
   "id":null,
   "mapping_method":"claim",
   "ldap":{
      "attributes":{
         "id":[
            "dn"
         ],
         "email":[
         ],
         "name":[
            "cn"
         ],
         "preferred_username":[
            "uid"
         ]
      },
      "insecure":true,
      "url":"$LDAP_URL"
   }
}
EOF
```

Verify the integration with the LDAP server for an admin user

1. Go to the cluster login page and check if there is an LDAP option for authentication in the list, if so click on this option.

2. Enter `rhoam-customer-admin` for _username_ and `Password1` (or whatever password you did set up when configuring LDAP) for _password_ and click the log in button.

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

9. Enter `rhoam-customer-admin` for _username_ and `Password1` (or whatever password you did set up when configuring LDAP) for _password_ and click the log in button.

   > You should be redirected to the 3scale main page.

Verify the integration with the LDAP server for a regular user

1. Go to the cluster login page and check if there is an LDAP option for authentication in the list, if so click on this option.

2. Enter `rhoam-test-user` for _username_ and `Password1` (or whatever password you did set up when configuring LDAP) for _password_ and click the log in button

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

8. Enter `rhoam-test-user` for _username_ and `Password1` (or whatever password you did set up when configuring LDAP) for _password_ and click the log in button.

   > You should be redirected to the 3scale main page.

**As the test is finished lets stop the EC2 instance.**

1. Access the AWS EC2 console, you can find the credentials in https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md

2. On the AWS EC2 console, right click on the LDAP Server EC2 instance and select stop instance.

   > Instance should change to stopped state.

   If having only AWS CLI access you can do the following

   ```bash
   EC2=`aws ec2 describe-instances --region eu-west-1 --filters "Name=tag:Name,Values=rhoam-ldap" --query 'Reservations[0].Instances[0].InstanceId' --output text`
   aws ec2 stop-instances --region eu-west-1 --instance-ids $EC2
   aws ec2 wait instance-stopped --instance-ids $EC2 --region eu-west-1

   ```
