---
automation:
  - INTLY-6408
components:
  - product-sso
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.6.0
estimate: 15m
---

# B08a - Verify User RHSSO Permissions

## Steps

The following steps should be tested for both rhmi developer and dedicated admin users

### Manage Realms in User RHSSO

1. Log into the User Red Hat Single Sign-On via the Solution Explorer
2. Hover over the Realm dropdown and Click **Add Realm**
3. Enter the following details in the realm creation form:
   - Name: Enter any name
   - Enabled: Switch to `On`
4. Click **Create**
   > Verify that the realm was created successfully. You should be redirected to the realm settings page of the realm you just created.
