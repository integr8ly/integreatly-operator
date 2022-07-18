---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.2.0
      - 1.6.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
      - 1.18.0
      - 1.21.0
      - 1.24.0
estimate: 15m
---

# A33 - Validate console links

## Prerequisites

- Logged in to a testing cluster as a test user (developer)

## Steps

**As a test user (developer)**

1. In OpenShift console, click on the dashboard icon on the top right corner
   > Validate that there are 3 links under OpenShift Managed Services: API Management, API Management Dashboards, API Management SSO
2. Click on `API Management` link and try to login using SSO IDP (testing-idp)
   > You should successfully log into 3scale
3. Back in OpenShift console, click on `API Management Dashboards` link and try to login using SSO IDP (testing-idp)
   > You should get 403 error
4. Back in OpenShift console, click on `API Management SSO` link and try to login using SSO IDP (testing-idp)
   > You should successfully log into User SSO

**As a customer admin user**

1. In OpenShift console, click on the dashboard icon on the top right corner
   > Validate that there are 3 links under OpenShift Managed Services: API Management, API Management Dashboards, API Management SSO
2. Click on `API Management` link and try to login using SSO IDP (testing-idp)
   > You should successfully log into 3scale
3. Back in OpenShift console, click on `API Management Dashboards` link and try to login using SSO IDP (testing-idp)
   > You should be redirected to Grafana
4. Search for "rate limiting" dashboard and try to access it
   > You should see the dashboard and data (and no errors)
5. Back in OpenShift console, click on `API Management SSO` link and try to login using SSO IDP (testing-idp)
   > You should successfully log into User SSO

**Validate RHOAM icons**

1. Open this url to see the current RHOAM icon https://issues.redhat.com/secure/projectavatar?pid=12324822. Validate that the same icon is used in following places in OpenShift console:
   > In OpenShift console, click on the dashboard icon on the top right corner -> see OpenShift Managed Services icons
   > Operators -> OperatorHub -> search for "rhoam"
