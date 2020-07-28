---
automation:
  - INTLY-7439
components:
  - product-3scale
environments:
  - osd-post-upgrade
estimate: 15m
targets:
  - 2.7.0
---

# H11 - Verify 3scale SMTP Configuration

## Description

Verify that Customer-admin can invite new users to 3scale via email by following the steps below and verify you have received an invitation email.

## Prerequisites

Login to 3scale console as a **customer-admin** (user with customer-admin permissions).

## Steps

1. Sign into the Openshift console as customer-admin
2. Home -> Projects -> redhat-rhmi-3scale
3. Networking -> Routes
4. Under Location, select the URL matching this pattern https://3scale-admin.apps.<your cluster>.s1.devshift.org
5. Sign into 3scale via single sign on option using customer-admin credentials
6. Click the gear icon in top right corner of console.
7. Users -> Invitations
8. Select +Invite a New Team Member
9. Put an email address you have access to into 'Send invitation to' input box and press 'Send'
10. Check your email account to see if you receive an email ensuring to check your spam/junk mail.
11. The email could take up to 80 minutes to arrive so recheck periodically.
