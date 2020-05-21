---
estimate: 45m
require:
  - G03
---

# G04 - Verify API management walkthrough

## Description

Protecting APIs using 3scale API Management Platform.

Important Note:

    Once you start wt4 and get to the 3scale part of it, you will see that test-user** does not have access to 'Add Product'.

    This is due to not having admin permissions. To resolve this you will need to login to 3scale in a separate browser as customer-admin. Once logged in as customer-admin.

        - select the gear icon in top right corner
        - select Users from left hand menu
        - select Listing from drop down menu
        - select the test-user you were using
        - under ADMINISTRATIVE change the role to Admin (full access)
        - select Update User

## Steps

1. Follow all steps in the solution-pattern.
   1. The Red Hat 3scale API Management Platform Dashboard shows the Audience and APIs panels.
   2. The i-greeting-integration-test-user\*\* API is visible and active in the PRODUCTS tab.
   3. The staging environment information URL in INTEGRATION/CONFIGURATION/ENVIRONMENST matches the pattern https://wt4-test-user10-3scale.apps.namespace.s1.devshift.org
   4. user_key=testkey? is shown INTEGRATION/CONFIGURATION/'STAGING ENVIRONMENT'/'Example curl for testing'
   5. The message 'Hello from, OpenShift appears in your Slack channel on completion of the walkthrough.
