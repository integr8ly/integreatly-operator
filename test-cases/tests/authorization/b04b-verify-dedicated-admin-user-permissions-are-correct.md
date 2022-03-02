---
automation:
  - INTLY-7748
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
estimate: 1h
tags:
  - per-release
  - automated
---

# B04B - Verify Dedicated Admin User Permissions are Correct

**Automated Test**: [user_dedicated_admin_permissions.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/user_dedicated_admin_permissions.go)

## Steps

The following steps are still not automated in [user_dedicated_admin_permissions.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/user_dedicated_admin_permissions.go). Once automated, the manual steps should be removed from this test case.

### View only permissions for RHMI Custom Resource

JIRA: [MGDAPI-3164](https://issues.redhat.com/browse/MGDAPI-3164)

1. Go to the **redhat-rhoam-operator** namespace in the OpenShift console
2. Go to **Home** > **Search**
3. Select **RHMI** in the dropdown
4. Verify that you are able to view the **rhoam** custom resource in the RHOAM operator namespace
5. Verify that the user doesn't have the permission to create a new RHOAM custom resource.
6. Click on the **rhoam** custom resource
7. Verify that you are able to view the RHOAM custom resource details and YAML file
8. Verify that you can't change the YAML by clicking it and ensuring it wont allow you to add or modify things, also verify there is not update button.
9. Verify that when you go to **Actions** > **Delete RHMI** that is greyed out and you can't click it.
