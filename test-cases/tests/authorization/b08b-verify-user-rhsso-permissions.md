---
automation:
  - INTLY-6408
components:
  - product-sso
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.4.0
      - 1.7.0
      - 1.10.0
      - 1.13.0
      - 1.16.0
estimate: 15m
---

# B08B - Verify User RHSSO Permissions

## Steps

The following steps should be tested for both rhmi developer and dedicated admin users

### Manage Realms in User RHSSO

1. Log into the User Red Hat Single Sign-On (Application Launcher -> Api Management SSO)
2. Hover over the Realm dropdown and Click **Add Realm**
3. Enter the following details in the realm creation form:
   - Name: Enter any name
   - Enabled: Switch to `On`
4. Click **Create**
   > Verify that the realm was created successfully. You should be redirected to the realm settings page of the realm you just created.
