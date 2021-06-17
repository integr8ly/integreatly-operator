---
automation:
  - MGDAPI-1920
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 1h
tags:
  - per-release
---

# H24 - Verify self-managed APIcast and custom policy

## Description

This test case should prove that it is possible for customers to deploy self-managed APIcast and use custom policies on it. The 3scale QE team will perform this test case in RHOAM each time there is an upgrade of 3scale. We (RHOAM QE) should only perform this if there are modifications on our end that might break the functionality - typically changes in permissions in RHOAM and/or OSD.
Additional context can be found in [MGDAPI-370](https://issues.redhat.com/browse/MGDAPI-370)

Note: in RHOAM v1.3.0 the [guide on this](https://access.redhat.com/documentation/en-us/red_hat_openshift_api_management/1/topic/a702e803-bbc8-47af-91a4-e73befd3da00) was published so it is preferred to follow the official guide and only use the text below as a supportive material.

The document includes the steps for manual test and instructions how to run the automation script

1. [Automation script](#Automation-script)
2. [Manual Test Steps](#Manual-Test-Steps)

## Steps

### Automation script

#### Automation Test procedure briefly

```sh
$ cd integreatly-operator/test/scripts/products/h24-verify-selfmanaged-apicast-and-custom-policy
$ test.sh
```

**Notes**
Parameters for configuration and recommendations for test could be found in _test.sh_ file and below:

#### Automation Test Prerequisites

- To be logged to Openshift cluster as admin (kubeadmin)

#### Recommended have the following before running the script

- Password for customer-admin user, if testing-idp is not yet available
- Customer Token, if testing-idp and customers are already available
  - To get customer token (example below is for user _customer-admin01_):
    - Login to OSD Portal as _customer-admin01_
    - at the top-right corner, under _Customer Admin 01_, select _Copy login command_ ->
      _Login with testing-idp_-> _Sign-In as customer-admin01 user/password_->_Display Token_
- Open 3scale Admin Portal, to be ready to put updates. Details will be notified by script.
- If re-executing the script, and if APIcast Configuration and Promotion in 3scale admin portal already done before, -
  have 3scale admin portal page opened or save user_key as appears in URL in Staging APIcast section.

#### Configuration parameters:

- create-testing-idp
  - _true_ - to allow script to create IDP and customer-admin users, if not exists yet.
  - _false_ - if IDP and customer-admin users do not exists yet  
    **Note:** IDP also could be created using separate idp creation script, as in following example:

```sh
  $ PASSWORD=<password> DEDICATED_ADMIN_PASSWORD=<password> scripts/setup-sso-idp.sh
```

- use-customer-admin-user
  - true - use this option
  - false - this option could be used for script debugging, using current user, such as kubeadmin
- promote-manually
  - true - use this option. The option will require manual configuration -  
    to use the self-managed APIcast instead of the built-in, and for promotion to Staging and Production.  
    Following notes are provided by the script in runtime, and script is waiting for these manual operations completion.

```
   *This is the manual step - Configure the service to use the self-managed APIcast instead of the built-in APIcast for API.
	*a. Navigate to 3scale Admin Portal. You can use the following command to find route: *oc get routes --namespace redhat-rhoam-3scale | grep admin*
	*b. In the Products section, click API → Integration → Settings → APIcast Self Managed
	*c. Change the Staging Public Base URL. Replace api-3scale-apicast- with selfmanaged-
	*d. Click Update Product.
	*e. Click API → Configuration → Promote to Staging and Promote to Production
	*f. Copy user_key value (from Staging APIcast - Example curl for testing) to command prompt:
```

- false - the option is in development, for fully automated script running in batch mode, without interaction with user
- 3scale-version="2.11"
- apicast-operator-version="v0.5.0"
- apicast-image-stream-tag="3scale2.11.0"
- apicast-namespace="selfmanaged-apicast"

#### Run the automation test script

```sh
$ cd integreatly-operator/test/scripts/products/h24-verify-selfmanaged-apicast-and-custom-policy
$ test.sh
```

- The script is working in interactive mode, two inputs will be required in command line in script runtime
  (while using following default parameters: _promote-manually=true_, _create-testing-idp=false_):
  - 1st input prompt:

```sh
    .......
    INFO[0009] Customer Login
    Enter Customer Admin user Token, and press Enter:
```

Get token as described in [Recommended have before...](#Recommended-have-the-following-before-running-the-script) section,
and put token to command prompt.

- 2nd input prompt:

```sh
  .......
    e. Click API → Configuration → Promote to Staging and Promote to Production
    f. Copy user_key value (from Staging APIcast - Example curl for testing) to command prompt:
```

Put **user_key** to command prompt. _User-key will be taken from 3scale portal, as described in Information provided by the Script in runtime, and in [configuration parameters](#Configuration-parameters:) section_

- Wait for the test to finish. Ensure that it finished successfully, Response code **200**, the end of the script output
  should look similar to following:

```sh
  INFO[0371] Create route for self-managed APIcast
  INFO[0372] routeHost: selfmanaged-staging.apps....
  INFO[0432] HTTP Request: https://selfmanaged-staging.....
  INFO[0432] Response Code: 200
  INFO[0432] Response Body: {
  "method": "GET",
   ........
  "HTTP_X_REQUEST_ID": "....",
  "HTTP_X_ENVOY_EXPECTED_RQ_TIMEOUT_MS": "15000"
  .......
  INFO[0432] Validation of Self-managed APIcast API gateway Deployment - Succeeded
  INFO[0432] Self-managed APIcast API gateway - Deployment script Completed
```

### Manual Test Steps

Unofficial - follow the official guide!

_1. This test case must be done using `customer-admin` user (or any other user from `dedicated-admins` group). Do not use `kubeadmin`!_

_2. Create a new namespace (e.g. selfmanaged-apicast) for self-managed APIcast_

- `oc new-project selfmanaged-apicast`
- be sure to use this project for the `oc` commands
- if you use different namespace, you will need to change a few commands below accordingly

_3. Import the image_

- `oc import-image 3scale-amp2/apicast-gateway-rhel8:<tag> --from=registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:<tag> --confirm`
- go to [catalog](https://catalog.redhat.com/software/containers/search), find the image and see the available tags
- typical tag name is `3scale<version>`, use the one that corresponds with the 3scale version installed on the cluster
- note the tag for later use (in step 7)
- if experiencing authorization error, follow step 4. and 5. Proceed with step 6. otherwise.

_4. (Skip if Step3 is ok) Create a secret with credentials to `registry.redhat.io`_

- `oc create secret docker-registry redhatio --docker-server=registry.redhat.io --docker-username=<username> --docker-password=<password> --docker-email=<email>`

_5. (Skip if Step3 is ok) Configure the pull secret_

- `oc secrets link default redhatio --for=pull`

_6. Clone the Example custom policy_

- `git clone git@github.com:3scale/apicast-example-policy.git`
- `cd apicast-example-policy`

_7. Edit the `openshift.yaml`_

- use the appropriate `<tag>` and potentially change the `selfmanaged-apicast` if you use different namespace

Replace

```
    strategy:
      type: Source
      sourceStrategy:
        from:
          kind: ImageStreamTag
          name: 'amp-apicast:${AMP_RELEASE}'
```

with

```
    strategy:
      type: Source
      sourceStrategy:
        from:
          kind: ImageStreamTag
          namespace: selfmanaged-apicast
          name: apicast-gateway-rhel8:<tag>
```

Replace

```
    nodeSelector: null
    output:
      to:
        kind: ImageStreamTag
        name: 'amp-apicast:latest'
```

with

```
    nodeSelector: null
    output:
      to:
        kind: ImageStreamTag
        namespace: selfmanaged-apicast
        name: apicast-gateway-rhel8:<tag>
```

Replace

```
    strategy:
      dockerStrategy:
        from:
          kind: ImageStreamTag
          name: 'amp-apicast:${AMP_RELEASE}'
```

with

```
    strategy:
      dockerStrategy:
        from:
          kind: ImageStreamTag
          namespace: selfmanaged-apicast
          name: apicast-gateway-rhel8:<tag>
```

_8. Create a new app_

- `oc new-app -f openshift.yml --param AMP_RELEASE=2.8`
- param value does not matter at this point but needs to be used since it is required parameter
- you can change make the parameter optional in `openshift.yaml` so you don't need to specify it at all

_9. Start custom policy builds_

- `oc start-build apicast-example-policy`
- Give some time between builds
- `oc start-build apicast-custom-policies`

_10. Get the 3scale Admin Portal token_

- navigate to the 3scale Admin Portal (web console) route can be got with `oc get routes --namespace redhat-rhoam-3scale | grep admin`
- log in as `customer-admin`
- navigate to `Account Settings/Personal/Tokens/Add Access Token`
- create an access token (full read and write access)

_11. Create an adminportal-credentials secret_

- `oc create secret generic adminportal-credentials --from-literal=AdminPortalURL=https://<access-token>@<url-to-3scale-admin-portal>`
- example: `oc create secret generic prod-sm-apicast --from-literal=AdminPortalURL=https://a926aaa5bb0ed9e89fd3a8c92bce7aefd5dc63748212db318eb968222fc45c7b@3scale-admin.apps.multiaz-24-trep.b5s6.s1.devshift.org`

_12. Use self-managed APIcast instead of the builded one for API (echo service)_

- navigate to 3scale Admin Portal (web console) route can be got with `oc get routes --namespace redhat-rhoam-3scale | grep admin`
- In `API's\Products` on the Dashboard screen go to
- API -> Integration -> Settings -> tick APIcast Self Managed radio-box
  - change "Staging Public Base URL" so that it is slightly different at the beginning, e.g. replace `api-3scale-apicast-` with `selfmanaged-`
  - Then use the `Update Product` button
- API -> Configuration -> Use the `Promote to Staging` and `Promote to Production` buttons

_13. Install "Red Hat Integration - 3scale APIcast gateway" operator via Operator Hub in OSD web console for the namespace_

- do not use community version!
- Accept the defaults and install in the selfmanaged-apicast namespac

_14. Create a self-managed APIcast_

- navigate to `Operators\Installed Operators` in osd
- Select `Red Hat Integration - 3scale APIcast gateway`
- Select the `APIcast` tab
- Use the `Create APIcast` button
- use `Form view`
- change "Admin Portal Credentials Ref" secret name to `adminportal-credentials`
- set "Image" to `image-registry.openshift-image-registry.svc:5000/selfmanaged-apicast/apicast-policy:example` (this also depends on the namespace name used)
- set "Configuration Load Mode" to boot
- Then use the `Create` button

_15. Create a route for the self-managed APIcast_

- navigate to the namespace of self managed apicast in OSD web console
- Networking -> Routes -> Create Route
  - name: whatever
  - hostname: must match the Staging Public Base URL (without protocol, without port)
  - service: select service of your apicast
  - target port: 8080 -> 8080
  - tick secure route, TLS termination: Edge
  - Then use the `Create` button

_16. Verify your work_

- navigate to 3scale Admin Portal -> API -> Configuration
- execute the curl for staging APIcast
- You should see exactly what you get when you do `curl https://echo-api.3scale.net` directly

_17. Make custom-policy available in 3scale Admin Portal_

- go back to apicast-example-policy GitHub repo
- `cat policies/example/0.1/apicast-policy.json` - note the output
- navigate to 3scale Admin Portal
- ? (Help) -> 3scale API docs -> APIcast Policy Registry Create
  - use `customer-admin` access token, i.e. the one generated in step 11.
  - name -> use name from apicast-policy.json
  - version -> use the version from apicast-policy.json
  - schema -> paste the `apicast-policy.json` content there
- press "Send Request" button
- navigate to 3scale Admin Portal -> API -> Integration -> Policies -> Add Policy
  - "APIcast Example Policy" should be in the list
