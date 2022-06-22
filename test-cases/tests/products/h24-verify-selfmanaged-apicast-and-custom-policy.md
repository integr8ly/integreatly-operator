---
automation:
  - MGDAPI-3146
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.19.0
      - 1.22.0
estimate: 1h
---

# H24 - Verify self-managed APIcast and custom policy

## Description

Note: This test case was automated and is executed automatically. However, the "custom policy" part is not yet automated so every third release comment out the [cleanup](https://github.com/integr8ly/integreatly-operator/blob/3b24a8d67fb0c2af8ca6502ff7bd593e69ad5bf2/test/common/selfmanaged_apicast.go#L125), run the h24 test and then do only the pieces required for setting up the custom policy.

This test case should prove that it is possible for customers to deploy self-managed APIcast and use custom policies on it. The 3scale QE team will perform this test case in RHOAM each time there is an upgrade of 3scale. We (RHOAM QE) should only perform this if there are modifications on our end that might break the functionality - typically changes in permissions in RHOAM and/or OSD.
Additional context can be found in [MGDAPI-370](https://issues.redhat.com/browse/MGDAPI-370)

Note: in RHOAM v1.3.0 the [guide on this](https://access.redhat.com/documentation/en-us/red_hat_openshift_api_management/1/topic/a702e803-bbc8-47af-91a4-e73befd3da00) was published so it is preferred to follow the official guide and only use the text below as a supportive material.

The document includes the steps for manual test and instructions how to run the automation script.
There are two automation options:

- Automation test could be executed in Pipeline as part of IDP based tests - **H24**. Test file: _integreatly-operator/test/common/selfmanaged_apicast.go_
- Automation test could run separately, by standalone script, in interactive mode and batch mode. Test directory: _integreatly-operator/test/scripts/products/h24-verify-selfmanaged-apicast-and-custom-policy/_  
  _Note. Automation tests do not include creation of custom policy_

1. [Automation Test in Pipeline](#Automation-Test-in-Pipeline)
2. [Standalon Automation script](#Standalon-Automation-script)
3. [Manual Test Steps](#Manual-Test-Steps)

## Steps

### Automation Test in Pipeline

Selfmanaged Apicast Automation test will be integrated into Pipeline, and also could be executed as e2e single test.
Test is a part of `IDP based tests`.

- Test name: `TestSelfmanagedApicast`
- Test location: `integreatly-operator/test/common`
- Test file: `selfmanaged_apicast.go`
- Test definition in `integreatly-operator/test/common/tests.go:`

```asciidoc
IDP_BASED_TESTS = []TestSuite{
    {
        []TestCase{
          .....
          {Description: "H24 - Verify selfmanaged Apicast", Test: TestSelfmanagedApicast},
	},
    .....
```

#### Configuration parameters

TestSelfmanagedApicast has following configuration parameters, that hardcoded as constants in `selfmanaged_apicast.go`

```
const (
  apicastOperatorVersion       = "0.5.0"
  apicastImageStreamTag        = "3scale2.11.0"
  apicastNamespace             = "selfmanaged-apicast"
  adminPortalCredentialsSecret = "adminportal-credentials"
  namespacePrefix              = "redhat-rhoam-"
  defaultDedicatedAdminName    = "customer-admin"
  promotionAttempts            = 20
  accountOrgName               = "Developer"
  serviceSystemName            = "api"
)
```

#### Run test as e2e single test

This is sample how to run TestSelfmanagedApicast as single test:

```
$ TEST=H24 INSTALLATION_TYPE=managed-api make test/e2e/single
```

Expected result could be similar to following:

```
...
time="2021-12-13T14:51:14+02:00" level=info msg="userKey found"
time="2021-12-13T14:51:14+02:00" level=info msg="Install 3scale APIcast gateway Operator"
time="2021-12-13T14:51:20+02:00" level=info msg="Create self-managed APIcast"
time="2021-12-13T14:51:20+02:00" level=info msg="Create route for self-managed APIcast"
time="2021-12-13T14:51:25+02:00" level=info msg="routeHost: selfmanaged-staging.apps.xxx.3bjq.s1.devshift.org"
....
time="2021-12-13T14:52:11+02:00" level=info msg="Response Code: 200"
time="2021-12-13T14:52:11+02:00" level=info msg="Validation of Self-managed APIcast deployment - Succeeded"
time="2021-12-13T14:52:11+02:00" level=info msg="Self-managed APIcast deployment - Completed"
....
PASS
ok  ...
```

### Standalon Automation script

#### Test location

```sh
$ cd integreatly-operator/test/scripts/products/h24-verify-selfmanaged-apicast-and-custom-policy
$ ls
h24-verify-selfmanaged-apicast-and-custom-policy.go  test.sh
```

**Notes**
Parameters for configuration and recommendations for test could be found in _test.sh_ file and below:

#### Automation Test Prerequisites

- To be logged to Openshift cluster as admin (kubeadmin)

#### Recommended have the following before running the script

- Customer Token
  - To get customer token (example below is for user _customer-admin01_):
    - Login to OSD Portal as _customer-admin01_
    - at the top-right corner, under _Customer Admin 01_, select _Copy login command_ ->
      _Login with testing-idp_-> _Sign-In as customer-admin01 user/password_->_Display Token_
- Open 3scale Admin Portal, to be ready to put updates. Details will be notified by script.
- If re-executing the script, and if APIcast Configuration and Promotion in 3scale admin portal already done before, -
  have 3scale admin portal page opened or save user_key as appears in URL in Staging APIcast section.

#### Configuration parameters:

- use-customer-admin-user
  - true - use this option
  - false - this option could be used for script debugging, using current user, such as kubeadmin
- promote-manually
  - false - to run script in batch mode. No interaction with user, and no any additional input from user required.
    The flow is similar to Test running in Pipeline.
  - true - the option could be useful for investigation. The option will require manual configuration -  
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

-apicast-operator-version="0.5.0" \
-apicast-image-stream-tag="3scale2.11.0" \
-apicast-namespace="selfmanaged-apicast" \
--namespace-prefix redhat-rhoam- \

#### Run the automation test script

```sh
$ cd integreatly-operator/test/scripts/products/h24-verify-selfmanaged-apicast-and-custom-policy
$ test.sh
```

- If the script is working in interactive mode, two inputs will be required in command line in script runtime
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

- If the script is running in batch mode, `-interactive-mode=false`, - no input required during script execution.
- Expected result will be similar to following:

```asciidoc
$ ./test.sh
....
project.project.openshift.io "selfmanaged-apicast" deleted
...
INFO[0026] Get 3scale admin Token
INFO[0026] Customer Login
Login successful.
...
3scale2.11.0
  tagged from registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:3scale2.11.0
...
Image Name:	apicast-gateway-rhel8:3scale2.11.0
....
INFO[0031] Get 3scale admin Portal
INFO[0031] Route: zync-3scale-provider-ndjgh
INFO[0031] 3scale Admin portal: 3scale-admin.apps.xxx.3bjq.s1.devshift.org
INFO[0031] Create adminportal-credentials secret
INFO[0032] Delete api-3scale-apicast- routes in redhat-rhoam-3scale namespace
INFO[0032] sendGetRequest, url: https://3scale-admin.apps.xxxx.3bjq.s1.devshift.org/admin/api/services.xml
INFO[0032] Service ID found: 2
INFO[0032] Service Update URL: https://3scale-admin.apps.xxxx.3bjq.s1.devshift.org/admin/api/services/2.xml
....
INFO[0038] sendGetRequest, url: https://3scale-admin.apps.xxxx.3bjq.s1.devshift.org/admin/api/accounts.xml
INFO[0038] Account ID found: 3
INFO[0038] sendGetRequest, url: https://3scale-admin.apps.xxxx.3bjq.s1.devshift.org/admin/api/accounts/3/applications.xml
INFO[0039] userKey found
INFO[0039] Install 3scale APIcast gateway Operator
INFO[0044] Create self-managed APIcast
INFO[0044] Create route for self-managed APIcast
INFO[0049] routeHost: selfmanaged-staging.apps.xxxx.3bjq.s1.devshift.org
Expected status 200 but got 503
Expected status 200 but got 503
INFO[0080] Response Code: 200
INFO[0080] Validation of Self-managed APIcast deployment - Succeeded
INFO[0080] Self-managed APIcast deployment - Completed
$
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
