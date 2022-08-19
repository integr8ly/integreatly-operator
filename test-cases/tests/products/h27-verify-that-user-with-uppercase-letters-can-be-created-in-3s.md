---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.4.0
      - 1.5.0
      - 1.8.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
      - 1.19.0
      - 1.22.0
      - 1.25.0
estimate: 30m
---

# H27 - Verify that user with uppercase letters can be created in 3scale

## Description

This test verifies that if there is an existing user with uppercase letters in the name in OpenShift, that user can be also created in 3scale (during RHOAM installation). The username in 3scale should be with lowercase letters

## Prerequisites

- OSD cluster with RHOAM installed
- Kubeadmin access to the OSD cluster
- Admin access to some github organization
- Github user with at least one uppercase letter in the username

## Steps

**Set up Github IDP for OSD cluster**

1. Register an openshift application by following this [guide](https://docs.openshift.com/container-platform/4.10/authentication/identity_providers/configuring-github-identity-provider.html#identity-provider-overview_configuring-github-identity-provider).

2. Grant the [application](https://github.com/settings/connections/applications) access to an org where you have admin access. Select the application and under orgnization access grant the application permissions to the org.

3. Retrieve and save the client secret and client id from the github [application](https://github.com/settings/developers).

4. Login to OCM with the token provided

```bash
ocm login --url=https://api.stage.openshift.com/ --token=<YOUR_TOKEN>
```

5. Set cluster name variable

```bash
CLUSTER_NAME="<CLUSTER_NAME>"
```

6. Get id of cluster and assign it to a variable

```bash
CLUSTER_ID=$(ocm get clusters --parameter search="display_name like '%$CLUSTER_NAME%'" | jq -r '.items[].id')
```

7. Set Client secret, id and org name variables (Values from step 3)

```bash
CLIENT_SECRET="<CLIENT_SECRET>"
CLIENT_ID="<CLIENT_ID>"
ORG_NAME="<ORG_NAME>"
```

8. Add GitHub IDP

```bash
ocm post https://api.stage.openshift.com/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/identity_providers --body=<<EOF
{
   "type":"GithubIdentityProvider",
   "name":"GitHub",
   "id":null,
   "mapping_method":"claim",
   "github":{
      "client_id":"$CLIENT_ID",
      "client_secret":"$CLIENT_SECRET",
      "organizations":[
         "$ORG_NAME"
      ]
   }
}
EOF
```

9. Log in to your cluster via Github IDP (go to OpenShift console, select Github IDP)
10. Verify that the user you've logged in with has an uppercase letters in its name

```bash
oc get users | awk '{print $1}' | grep -i <your-username>
```

11. In OpenShift console (when logged in as your github user), select the launcher on the top right menu -> API Management -> Github IDP and log in to 3scale
    > Verify that you can successfully log in
12. Go to Account settings (top right menu) -> Personal -> Personal Details
    > Verify that your username contains only lowercase letters
13. Change some letter in your username to uppercase letter (e.g. myuser -> Myuser) and confirm the change
    > Verify that after RHOAM operator reconciles (~5 minutes), your username is changed back to lowercase letters
