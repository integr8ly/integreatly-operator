---
automation:
  - INTLY-6652
components:
  - product-codeready
environments:
  - osd-post-upgrade
targets:
  - 2.4.0
  - 2.7.0
---

# B05 - Verify Codeready CRUDL Permissions

**Automated Test**: [codeready_crudl.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/codeready_crudl.go)

## Steps

The following steps are still not automated [codeready_crudl.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/codeready_crudl.go). Once automated, the manual steps should be removed from this test case.

### Users should only manage their own Codeready workspaces

JIRA: [INTLY-6652](https://issues.redhat.com/browse/INTLY-6652)

1. Log into Codeready as a RHMI developer user
   > Verify that you cannot see workspaces created by the other users
