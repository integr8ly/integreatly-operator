---
products:
  - name: rhoam
tags:
  - automated
---

# H30 - Verify that product can be created via product CR

## Description

This test case should allow an admin user to create a product.

## Steps

1. Verify 3scale is cluster scoped.
2. Make a project.
3. Get the admin URL.
4. Get admin token, log in details and sign into 3scale.
5. Create a secret to be used when creating the product.
6. Create product cr using the same namespace.
7. Verify it has been created.
