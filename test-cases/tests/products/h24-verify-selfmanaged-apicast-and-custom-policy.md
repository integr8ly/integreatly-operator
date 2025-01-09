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
      - 1.25.0
      - 1.28.0
      - 1.31.0
      - 1.35.0
      - 1.38.0
      - 1.41.0
estimate: 1h
---

# H24 - Verify self-managed APIcast and custom policy

## Description

Note: Creation of self-managed APIcast has been automated. However, the "custom policy" part is not yet automated so every third release run the H24 test without cleanup and proceed with the stepr below for setting up the custom policy. Before running the H24 test take a look [here at apicastOperatorVersion](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/selfmanaged_apicast.go#L48) and compare with current version. It can be seen by navigating to Operator Hub in OSD and searching for "Red Hat Integration - 3scale APIcast gateway". After clicking on the tile check the "Latest version" there. If there is not a match then update it in the code (please create a PR and/or ping the QE coordinator also). Execute the following to actually run the test to get your self-managed APIcast ready:

```
LOCAL=false SKIP_CLEANUP=true TEST=H24 INSTALLATION_TYPE=managed-api make test/e2e/single
```

Note: if this fails with some issue related to the version change just revert the changes and run with what was there originally. Check what the version actually is in CSV once the APIcast operator is installed. It should contain `-mas`.

Note: the templates for custom policies are no longer supported so the [RHOAM guide](https://access.redhat.com/documentation/en-us/red_hat_openshift_api_management/1/topic/a702e803-bbc8-47af-91a4-e73befd3da00) needs to be updated. Until that happens see the [official 3scale guide](https://access.redhat.com/documentation/en-us/red_hat_3scale_api_management/2.12/html/installing_3scale/installing-apicast#injecting-custom-policies-with-the-apicast-operator) for more information instead.

Note:

This test case should prove that it is possible for customers to deploy self-managed APIcast and use custom policies on it. Additional context can be also found in [MGDAPI-370](https://issues.redhat.com/browse/MGDAPI-370)

## Steps

### Run test as e2e single test

This is a way to run TestSelfmanagedApicast as single test:

```
LOCAL=false SKIP_CLEANUP=true TEST=H24 INSTALLATION_TYPE=managed-api make test/e2e/single
```

If finished successfully the self-managed APIcast should be up and running in selfmanaged-apicast namespace. Also the 3scale Product (called `h24_test_product_<number>`) should be present using the self-managed APIcast under the hood. What is left to do is to apply and validate custom policy.

### Custom Policy Manual Test Steps

Based on [official documentation](https://access.redhat.com/documentation/en-us/red_hat_3scale_api_management/2.12/html/installing_3scale/installing-apicast#injecting-custom-policies-with-the-apicast-operator). The steps below are meant to work on their own, there should not be a need to read the guide while following them.

_1. These steps must be done using `customer-admin` user (or any other user from `dedicated-admins` group). Do not use `kubeadmin`!_

_2. Create a secret in selfmanaged-apicast namespace_

- clone [Ngx example policy](https://github.com/3scale/APIcast/tree/master/examples/policies/ngx-example/1.0.0) or [Apicast example policy](https://github.com/3scale-qe/apicast-example-policy/tree/master/policies/example/0.1)

> Ngx is preferred and the steps below assume using it.

```
oc create secret generic ngx-custom-policy-secret \
 --from-file=./apicast-policy.json \
 --from-file=./init.lua \
 --from-file=./ngx_example.lua
```

_3. Update APIcast CR_

- edit the existing APIcast CR in selfmanaged-apicast namespace created by H24 test.
- `oc edit apicast example-apicast -n selfmanaged-apicast`
- add the block below to `spec`:

```
  customPolicies:
    - name: 'Ngx example policy'
      version: '1.0.0'
      secretRef:
        name: ngx-custom-policy-secret
```

- change the `configurationLoadMode` to `lazy` in `spec`.

_4. Get the 3scale Admin Portal token_

- navigate to the 3scale Admin Portal (web console), route can be got with `oc get routes --namespace redhat-rhoam-3scale | grep admin`
- login as `customer-admin`
- navigate to `Account Settings/Personal/Tokens/Add Access Token`
- create an access token (full read and write access)

_5. Make custom-policy available in 3scale Admin Portal_

- see [apicast-policy.json](https://github.com/3scale/APIcast/tree/master/examples/policies/ngx-example/1.0.0/apicast-policy.json)
- navigate to 3scale Admin Portal
- ? (Help) -> 3scale API docs -> `APIcast Policy Registry Create` API endpoint
  - use `customer-admin` access token, i.e. the one generated in step above.
  - name -> use name from apicast-policy.json
  - version -> use the version from apicast-policy.json
  - schema -> paste the `apicast-policy.json` content there
- press "Send Request" button
- navigate to 3scale Admin Portal -> Products -> (select h24_test_product) -> Integration -> Policies -> Add policy
  - "Ngx example policy" should be in the list
- add the policy (click on it) and configure it (click on it again) to use some header and value (e.g. `TEST_HEADER` and `test-value`)
- update the policy chain

_6. Verify your work_

- navigate to 3scale Admin Portal -> API -> Configuration
- promote to staging and production
- restart the selfmanaged apicast pod to apply the changes
- execute the curl for staging APIcast
- You should see what you get when you do `curl https://echo-api.3scale.net` directly
- You should see the additional `TEST_HEADER` being added by Ngx example policy

### Troubleshooting

- make sure the changes are promoted to staging and production
- restart (delete) the APIcast pod
- look at APIcast pod log for errors
- check the APIcast CR yaml if it contains the required config as written above
- value in `.spec.customPolicies.<your-policy>.name` must match with value in `apicast-policy.json`
- check the policy, policy configuration and the policy chain in 3scale Admin Portal
- you can do the above also using `Proxy Policies Chain Show` API endpoint - to bypass the UI in case of a bug there
- if there are multiple `h24_test_product` products, remove all the products except the first one created (lowest ID)
