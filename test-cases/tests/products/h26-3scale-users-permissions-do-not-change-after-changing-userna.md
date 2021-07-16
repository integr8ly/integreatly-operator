---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.4.0
      - 1.9.0
estimate: 15m
tags:
  - manual-selection
---

# H26 - 3scale user's permissions do not change after changing username in 3scale UI

## Description

When user modifies a username in 3scale UI, the new username gets deleted and original name recreated after integreatly-operator reconciles.
Verify that the original user permissions (admin) are not changed after the username gets recreated

## Prerequisites

- SSO IDP configured on a cluster
- passwords for customer-admin and test-user users

## Steps

1. In browser window, login to OpenShift console as `customer-admin03`
2. In anonymous browser window, login to OpenShift console as `test-user03`
3. In both browser windows, select the launcher on the top right menu -> API Management -> testing-idp and login to 3scale
4. In browser window (as customer-admin), go to Account settings (top right menu) -> Users -> Listing
5. Click on the `test-user03`, change the role to `Admin (full access)` -> click on Update User
6. As a `test-user03`, go to Account settings (top right menu) -> Personal -> Personal Details
7. Change Username to `test-user03-changed` -> click on `Update Details`
8. Wait for integreatly operator to reconcile (it could take ~5 minutes) and wait for the `test-user03-changed` username to change back to `test-user03`
   > Verify that the Role for `test-user03` is still admin (haven't changed)
