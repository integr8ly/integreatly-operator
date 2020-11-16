---
# See the metatadata section in the README.md for details on the
# allowed fields and values
components:
  - product-3scale
  - product-sso
estimate: 1h
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.0.0
---

# B09 - Validate Authentication with LDAP IDP

## Description

We want to validate that RHOAM is able to authenticate to an LDAP server via IDP.

## Steps

First lets start the AWS EC2 instance that have the LDAP service installed.

1. Access the AWS EC2 console, you can find the credentials in https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md

2. On the AWS EC2 console, right click on the LDAP Server EC2 instance and select start instance.

   > Instance should change to running state

3. Once the instance is in a running state verify if the LDAP service is up and running.

   ```bash
   // you need to change the <EC2 instance IP> to the EC2 instance IP

   curl -vv "ldap://<EC2 instance IP>:389/dc=ec2-3-133-150-27,dc=us-east-2,dc=compute,dc=amazonaws,dc=com?uid?sub?(uid=rhoam)"

   DN: uid=rhoam,ou=people,dc=ec2-3-133-150-27,dc=us-east-2,dc=compute,dc=amazonaws,dc=com
    uid: rhoam
   ```

Create the integration with RHOAM via IDP.

1. On the OCM console https://qaprodauth.cloud.redhat.com/openshift, go to the cluster you want to add the IDP.

2. Click on the `Access Control` tab.

3. Press the `Add identity provider` button.

4. A form will pop up in a modal with three sections:

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

As the test is finished lets stop the EC2 instance.

1. Access the AWS EC2 console, you can find the credentials in https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md

2. On the AWS EC2 console, right click on the LDAP Server EC2 instance and select stop instance.

   > Instance should change to stopped state.
