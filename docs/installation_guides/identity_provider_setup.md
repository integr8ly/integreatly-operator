# Set up testing IDP for OSD cluster
You can use the `scripts/setup-sso-idp.sh` script to setup a "testing-idp" realm in a cluster SSO instance and add it as IDP of your OSD cluster.
With this script you will get few regular users - test-user[01-10] and few users that will be added to dedicated-admins group - customer-admin[01-03].

## Prerequisites
- `oc` command available on your machine (the latest version can be downloaded [here](https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/))
- `ocm` command available ( the newest CLI can be downloaded [here](https://github.com/openshift-online/ocm-cli/releases) and you install it with `mv (your downloaded file) /usr/local/bin/ocm`) (necessary only if using OSD cluster)
- OC session with cluster admin permissions in a target cluster
- OCM session (necessary only if using OSD cluster)
- `openssl` command available on your machine 

| Variable                  | Format  | Type     | Default        | Details                                                                     |
|---------------------------|---------|:--------:|----------------|-----------------------------------------------------------------------------|
| PASSWORD                  | string  | Optional | _None_         | If empty, a random password is generated for the testing users.             |
| DEDICATED_ADMIN_PASSWORD  | string  | Optional | _None_         | If empty, a random password is generated for the testing dedicated admins.  |
| REALM                     | string  | Optional | testing-idp    | Set the name of the realm in side cluster sso                               |
| REALM_DISPLAY_NAME        | string  | Optional | Testing IDP    | Realm display name in side cluster sso                                      |
| INSTALLATION_PREFIX       | string  | Optional | _None_         | If empty, the value is gotten for the the cluster using `oc get RHMIs --all-namespaces -o (pipe) jq -r .items[0].spec.namespacePrefix` |
| ADMIN_USERNAME            | string  | Optional | customer-admin | Username prefix for dedicated admins                                        |
| NUM_ADMIN                 | int     | Optional | 3              | Number of dedicated admins to be set up                                     |
| REGULAR_USERNAME          | string  | Optional | test-user      | Username prefix for regular test users                                      |
| NUM_REGULAR_USER          | int     | Optional | 10             | Number of regular user to be used.                                          |

## Configuring Github OAuth

*Note:* Following steps are only valid for OCP4 environments and will not work on OSD due to the Oauth resource being periodically reset by Hive.

Follow [docs](https://docs.openshift.com/container-platform/4.1/authentication/identity_providers/configuring-github-identity-provider.html#identity-provider-registering-github_configuring-github-identity-provider) on how to register a new Github Oauth application and add the necessary authorization callback URL for your cluster as outlined below:

```
https://oauth-openshift.apps.<cluster-name>.<cluster-domain>/oauth2callback/github
```

Once the Oauth application has been registered, navigate to the Openshift console and complete the following steps:

*Note:* These steps need to be performed by a cluster admin

- Select the `Search` option in the left-hand nav of the console and select `Oauth` from the "Resources" dropdown
- A single Oauth resource should exist named `cluster`, click into this resource
- Scroll to the bottom of the console and select the `Github` option from the `add` dropdown
- Next, add the `Client ID` and `Client Secret` of the registered Github Oauth application
- Ensure that the Github organization from where the Oauth application was created is specified in the Organization field
- Once happy that all necessary configurations have been added, click the `Add` button
- For the validation purposes, log into the Openshift console from another browser and check that the Github IDP is listed on the login screen

## Set up dedicated admins

To setup your cluster to have dedicated admins run the `./scripts/setup-htpass-idp.sh` script which creates htpasswd identity provider and creates users.