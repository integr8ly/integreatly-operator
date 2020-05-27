---
tags:
  - happy-path
require:
  - N01
estimate: 30m
---

# N04 - There are no failures with minimal CIDR block

## Prerequisites

Login as **kube-admin** or user with **admin** role

## Steps

1. BYOC cluster should not use basic network settings, instead it should use the following settings

- Machine CIDR: 10.11.128.0/23
- Service CIDR: 10.11.0.0/18
- Pod CIDR: 10.11.64.0/18
- Host Prefix: 23

2. Perform 2.1 installation, CRO will begin to fail when provisioning the SSO Postgres instance
3. Perform upgrade to 2.1.1
4. Ensure installation completes successfully
5. Ensure no alerts are firing
6. Login to all products
7. Execute all walkthroughs

Reference: https://issues.redhat.com/browse/INTLY-7658
