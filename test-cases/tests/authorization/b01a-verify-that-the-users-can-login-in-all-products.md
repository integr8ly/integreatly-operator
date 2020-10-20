---
automation:
  - INTLY-7416
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - rhpds
estimate: 15m
tags:
  - per-release
---

# B01a - Verify that the users can login in all products

## Description

Products:

- OpenShift Console
- Fuse (redhat-rhmi-fuse)
- 3Scale (redhat-rhmi-3scale), route name: zync-3scale-provider, URL starting with "https://3scale-admin"
- AMQ Online (redhat-rhmi-amq-online), route name: console
- Codeready workspaces (redhat-rhmi-codeready), route name: codeready
- User SSO (redhat-rhmi-user-sso), route name: keycloak-edge
- UPS (redhat-rhmi-ups)
- Solution explorer (redhat-rhmi-solution-explorer)
- Apicurito (redhat-rhmi-apicurito), route name: apicurito

## Steps

1. Login to all Products listed in the Description using a **developer** user
   > Should succeed
2. Try to login to the Cluster SSO using a **developer** user
   > Should fail
3. Login to all Products listed in the Description using a **dedicated-admin** user
   > Should succeed
4. Try to login to the Cluster SSO using a **dedicated-admin** user
   > Should fail
