---
automation:
  - MGDAPI-1262
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.1.0
      - 1.6.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
estimate: 15m
---

# A32 - Validate SSO URLs from RHMI CR

## Prerequisites

- Logged in to a testing cluster as a `kubeadmin`

## Steps

**Validate RHSSO URL from RHOAM CR**

1. Run following command and note down the password from the ouput

```
oc get secret credential-rhsso -n redhat-rhoam-rhsso -o json | jq -r '.data.ADMIN_PASSWORD' | base64 --decode
```

2. Open the RHSSO admin route:

```
open $(oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r .status.stages.authentication.products.rhsso.host)
```

3. Click on Administration console
4. Log in with user: `admin`, password: `<from-previous-command>`
   > The login should succeed

**Validate USER-SSO URL from RHOAM CR**

1. Open the User SSO route:

```
open $(oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r .status.stages.products.products.rhssouser.host)
```

3. Click on Administration console
4. Log in using testing-idp user (customer-admin01/Password1)
   > The login should succeed
