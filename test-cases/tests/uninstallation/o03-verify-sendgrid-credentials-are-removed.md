---
automation:
  - INTLY-9992
products:
  - name: rhoam

tags:
  - automated
---

# O03 - Verify SendGrid credentials are removed

## Description

This test case should verify that the OCM SendGrid Service has correctly removed the SendGrid sub account for the cluster.

## Prerequisites

- The ID of the OpenShift cluster from OCM e.g. `1f3ufk5dn7suk8m4um8op9k8le9gi64h`, referred to as `<cluster-id>` in this test case
- SendGrid API key for the staging account, referred to as `<sendgrid-api-key>` in this test case
- The OpenShift cluster has been deleted

## Steps

1. Ensure a user named after the cluster ID doesn't exist in the SendGrid account

   ```
   curl -ks -H 'Authorization: Bearer <sendgrid-api-key>' https://api.sendgrid.com/v3/subusers | jq -r '.[] | select(.username=="<cluster-id>")'
   ```

   > Empty output
