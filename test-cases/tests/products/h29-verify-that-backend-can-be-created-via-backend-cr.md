---
products:
  - name: rhoam
tags:
  - automated
---

# H29 - Verify that backend can be created via backend CR

## Description

This test case should allow an admin user to create a backend.

## Steps

1. Verify 3scale is cluster scoped.
2. Make a project.
3. Get the admin URL.
4. Get admin token, log in details and sign into 3scale.
5. Create a secret to be used when creating the backend.
6. Create a backend cr under the same namespace.
7. Verify it has been created.
