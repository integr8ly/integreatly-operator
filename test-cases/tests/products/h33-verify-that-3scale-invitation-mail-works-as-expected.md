---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
tags:
  - manual-selection
---

# H33 - Verify that 3scale invitation mail works as expected

## Description

Verify that 3scale invitation mail is sent out

## Steps

1. Log in as `customer-admin01`

2. Navigate to API management in the quick access menu at the top of the console page.

3. From the drop-down menu at the top navigate to Dashboard > Accounts > Developer (John Doe) > Invitations

4. Click the invite user icon and input your email.

5. You should receive an email with an invitation (This may take a few minutes).
