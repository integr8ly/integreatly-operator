---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
tags:
  - manual-selection
---

# H32 - Verify that 3scale invitation mail is set correctly

## Description

3scale invitation mail is valid and present

## Steps

1. log in as `customer-admin01`

2. Navigate to API management in the quick access menu at the top of the console page.

3. From the drop-down menu at the top navigate to Audience > Developer Portal > Domain & Access

4. Verify that the outgoing email is `noreply-alert@rhmw.io`

5. Change the outgoing email and update the account.

6. Verify that the outgoing email is reverted to `noreply-alert@rhmw.io`
