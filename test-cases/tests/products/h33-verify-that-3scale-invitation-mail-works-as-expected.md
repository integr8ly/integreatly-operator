---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.21.0
      - 1.24.0
      - 1.27.0
      - 1.30.0
      - 1.33.0
      - 1.36.0
      - 1.39.0
      - 1.42.0
estimate: 10m
---

# H33 - Verify that 3scale invitation mail works as expected

## Description

Verify that 3scale invitation mail is sent out

## Steps

1. Log in as `customer-admin01`

2. Navigate to API management in the quick access menu at the top of the console page.

3. From the Dashboard navigate to ACCOUNTS > Developer (John Doe) > Invitations

4. Click the invite user icon and input your email.

5. You should receive an email with an invitation (This may take a few minutes).
