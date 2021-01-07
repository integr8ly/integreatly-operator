---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
estimate: 2h
tags:
  - manual-selection
---

# H24 - Verify self-managed APIcast and custom policy

## Description

This test case should prove that it is possible for customers to deploy self-managed APIcast and use custom policies on it. The 3scale QE team will perform this test case in RHOAM each time there is an upgrade of 3scale. We (RHOAM QE) should only perform this if there are modifications on our end that might break the functionality - typically changes in permissions in RHOAM and/or OSD.
Additional context can be found in [MGDAPI-370](https://issues.redhat.com/browse/MGDAPI-370)

## Steps

_1. This test case must be done using `customer-admin` user (or any other user from `dedicated-admin` group). Do not use `kubeadmin`!_
_2. Create a new namespace (e.g. selfmanaged-apicast) for self-managed APIcast_

- `oc project selfmanaged-apicast`
- be sure to use this project for the `oc` commands

_3. Create a secret with credentials to `registry.redhat.io`_

- `oc create secret docker-registry redhatio --docker-server=registry.redhat.io --docker-username=<username> --docker-password=HIDDEN --docker-email=<email>`

_4. Configure the pull secret_

- `oc secrets link default redhatio --for=pull`

_5. Import the image_

- `oc import-image 3scale-amp2/apicast-gateway-rhel8:3scale<version> --from=registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:<tag> --confirm`
- go to [catalog](https://catalog.redhat.com/software/containers/search), find the image and see the available tags
- typical tag name is `3scale<version>`, use the one that corresponds with the 3scale version installed on the cluster
- note the tag for later use (in step 7)

_6. Clone the Example custom policy_

- `git clone git@github.com:3scale/apicast-example-policy.git`
- `cd apicast-example-policy`

_7. Edit the `openshift.yaml`_

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
      dockerfile: |
        FROM scratch
        COPY . src
```

with

```
      dockerfile: |
        FROM scratch
        COPY . src
        USER root
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
- param value does not matter at this point

_9. Start custom policy builds_

- `oc start-build apicast-example-policy`
- `oc start-build apicast-custom-policies`
- once finished, the `image-registry.openshift-image-registry.svc:5000/selfmanaged-apicast/apicast-policy:example` should be available to be used as image for self-managed APIcast

_10. Install "Red Hat Integration - 3scale APIcast gateway" operator via Operator Hub in OSD web console for the namespace_

- do not use community version!

_11. Get the 3scale Admin Portal token_

- navigate to the 3scale Admin Portal (web console)
- log in as `customer-admin`
- create an access token (full read and write access)

_12. Create an adminportal-credentials secret_

- `oc create secret generic adminportal-credentials --from-literal=AdminPortalURL=https://<access-token>@<url-to-3scale-admin-portal>`
- example: `oc create secret generic prod-sm-apicast --from-literal=AdminPortalURL=https://a926aaa5bb0ed9e89fd3a8c92bce7aefd5dc63748212db318eb968222fc45c7b@3scale-admin.apps.multiaz-24-trep.b5s6.s1.devshift.org`

_13. Create a self-managed APIcast_

- navigate back to self-managed APIcast operator and click on "Create an APIcast"
- use `Edit Form`
- change "Admin Portal Credentials Ref" secret name to `adminportal-credentials`
- set "Configuration Load Mode" to lazy
- set "Image" to `image-registry.openshift-image-registry.svc:5000/selfmanaged-apicast/apicast-policy:example`

_14. Use self-managed APIcast instead of the builded one for API (echo service)_

- navigate to 3scale Admin Portal (web console)
- API -> Integration -> Settings -> tick APIcast Self Managed radiobox
  - change "Staging Public Base URL" so that it is slightly different at the beginning, e.g. replace `api-3scale-apicast-` with `selfmanaged-`
- API -> Configuration -> Promote to Staging and to Production

_15. Create a route for the self-managed APIcast_

- navigate to the namespace of self managed apicast in OSD web console
- Networking -> Routes -> Create Route
  - name: whatever
  - hostname: must match the Staging Public Base URL (without protocol, without port)
  - service: select service of your apicast
  - target port: 8080 -> 8080
  - tick secure route, TLS termination: Edge

_16. Verify your work_

- navigate to 3scale Admin Portal -> API -> Configuration
- execute the curl for staging APIcast
- You should see exactly what you get when you do `curl https://echo-api.3scale.net` directly

_17. Make custom-policy available in 3scale Admin Portal_

- go back to apicast-example-policy GitHub repo
- `cat policies/example/0.1/apicast-policy.json` - note the output
- navigate to 3scale Admin Portal
- ? (Help) -> 3scale API docs -> APIcast Policy Registry Create
  - use `customer-admin` access token
  - name -> use name from apicast-policy.json
  - schema -> paste the `apicast-policy.json` content there
- press "Send Request" button
- navigate to 3scale Admin Portal -> API -> Integration -> Policies -> Add Policy
  - "APIcast Example Policy" should be in the list
