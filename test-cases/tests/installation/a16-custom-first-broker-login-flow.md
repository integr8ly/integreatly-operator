---
products:
  - name: rhmi
  - name: rhoam
tags:
  - automated
---

# A16 - Custom first broker login flow

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/authdelay_first_broker_login.go

## Description

The custom first broker login flow is used by both 3Scale and Codeready. This test case guides you through verifying this auth flow via 3Scale. Once this works with 3Scale, it should also work for Codeready as both are using the same flow. No further tests for Codeready is required.

## Prerequisites

Login must be carried out with a user that has never logged into a testing cluster previously.

- To check if a user has not been used to login yet, run `oc get users`. Any users that are not listed in the results have not been used to log in yet.

Integreatly-operator needs to be running

## Steps

1. Open the 3Scale console "https://3scale-admin.apps.\<cluster-subdomain>"

   - Login as kubeadmin via oc cli
   - Run the following command to get the 3scale console route:

     `oc get route -n redhat-rhmi-3scale --selector=zync.3scale.net/route-to=system-provider`

2. Login to the 3Scale console

   - Click `Authenticate through Red Hat Single Sign-On`
   - Login via the testing idp

3. You should then get redirected from the IDP to the OpenShift OAuth and then to the cluster SSO.

   - You should see a screen saying `Your account is being provisioned` (this page gets refreshed automatically)

4. You should then be redirected to the 3Scale console (this may take a minute or two).
   - Once redirected, you should be successfully logged in.
