---
estimate: 45m
components:
  - product-fuse
targets: []
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

1. Login to the Solution Explorer as `test-user-XX` and confirm that `All services` tab is selected by default.
2. Make sure that you can complete all of the walkthrough steps and that they are accurate, without any spelling mistakes and up to date
