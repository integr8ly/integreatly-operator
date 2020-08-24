---
# See the metatadata section in the README.md for details on the
# allowed fields and values
automation:
  - INTLY-9950
environments:
  - osd-fresh-install
estimate: 30m
tags:
  - per-release
---

# O03 - Verify SendGrid credentials are removed

## Description

This test case should verify that the OCM SendGrid Service has correctly removed the SendGrid sub account for the cluster.

## Prerequisites

- The ID of the OpenShift cluster from OCM e.g. `1f3ufk5dn7suk8m4um8op9k8le9gi64h`
- SendGrid credentials for the SendGrid account
- The OpenShift cluster has been deleted

## Steps

1. Login to [SendGrid](https://app.sendgrid.com/login/)

2. Go to _Settings -> Subuser Management_

3. Search for the Cluster ID, ensure it doesn't exist
