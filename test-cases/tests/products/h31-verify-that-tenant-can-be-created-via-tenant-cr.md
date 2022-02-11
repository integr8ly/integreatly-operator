---
products:
  - name: rhoam
tags:
  - automated
---

# H31 - Verify that tenant can be created via tenant CR

## Description

This test case should allow a master user to create a tenant.

## Steps

1. Verify 3scale is cluster scoped.
2. Create a project.
3. Create a secret.
4. Get master URL.
5. Create a tenant cr under the same namespace.
6. Get master token, login details and sign into 3scale.
7. Verify tenant was created.
