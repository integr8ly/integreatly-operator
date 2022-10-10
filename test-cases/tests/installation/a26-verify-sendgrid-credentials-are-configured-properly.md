---
products:
  - name: rhoam
automation:
  - INTLY-9992
tags:
  - automated
---

# A26 - Verify SendGrid credentials are configured properly

## Description

This test case should verify that the OCM SendGrid Service has correctly setup SMTP credentials on the OpenShift cluster.

## Steps

1. Retrieve the SendGrid SMTP credentials from the RHMI Operator namespace

   ```
   # SMTP username, base64 encoded. This will be referred to as <username> from now on.
   oc get secret redhat-rhmi-smtp -n redhat-rhmi-operator -o jsonpath='{.data.username}'

   # SMTP password, base64 encoded. This will be referred to as <password> from now on.
   oc get secret redhat-rhmi-smtp -n redhat-rhmi-operator -o jsonpath='{.data.password}'
   ```

2. Telnet to the SMTP service to ensure the credentials are correct

   ```
   # Establish connection
   telnet $(oc get secret redhat-rhmi-smtp -n redhat-rhmi-operator -o jsonpath='{.data.host}' | base64 --decode) $(oc get secret redhat-rhmi-smtp -n redhat-rhmi-operator -o jsonpath='{.data.port}' | base64 --decode)

   # Once connection is established, "service ready at" message is shown
   EHLO

   # Once EHLO has completed, attempt auth
   auth login

   # When you see "VXNlcm5hbWU6" (base64 encoded "Username:")
   <username>

   # When you see "UGFzc3dvcmQ6" (base64 encoded "Password:")
   <password>
   ```

   > "Authentication successful" is shown

3. End the Telnet session, enter `Ctrl + ]` and then `Ctrl + d`
