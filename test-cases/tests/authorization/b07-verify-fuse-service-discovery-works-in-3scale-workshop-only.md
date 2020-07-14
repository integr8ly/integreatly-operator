---
tags:
  - 3scale
  - fuse
---

# B07 - Verify Fuse Service Discovery Works in 3Scale (Workshop Only)

This test should only be run in a RHPDS environment.

## Steps

1. Login to Fuse Online as a RHMI developer user
2. Create a new dummy integration in Fuse Online (you can create an API provider and define a dummy path)
3. Publish the integration
4. Login to 3Scale dashboard as a RHMI developer user
5. Click **New Product**
6. Select **Import from OpenShift**
   > Verify that the `Name` field is populated with the name of your dummy Fuse integration
7. Click **Create Product**
   > Verify that the product was created successfully
