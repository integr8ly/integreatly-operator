---
automation:
  - INTLY-7416
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
estimate: 15m
tags:
  - per-release
---

# B01B - Verify that the users can login in all products

## Description

Products:

- OpenShift Console
- 3Scale (redhat-rhoam-3scale), route name: zync-3scale-provider, URL starting with "https://3scale-admin"
- User SSO (redhat-rhoam-user-sso), route name: keycloak-edge

## Steps

1. Login to all Products listed in the Description using a **developer** user
   > Should succeed
2. Try to login to the Customer Grafana using a **developer** user
   > Should fail
3. Try to login to the Cluster SSO using a **developer** user
   > Should fail
4. Login to all Products listed in the Description using a **dedicated-admin** user
   > Should succeed
5. Try to login to the Customer Grafana using a **dedicated-admin** user
   > Should succeed
6. Try to login to the Cluster SSO using a **dedicated-admin** user
   > Should fail
