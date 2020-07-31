---
automation:
  - INTLY-7748
environments:
  - osd-post-upgrade
  - osd-fresh-install
estimate: 1h
tags:
  - per-release
---

# B04 - Verify Dedicated Admin User Permissions are Correct

**Automated Test**: [user_dedicated_admin_permissions.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/user_dedicated_admin_permissions.go)

## Steps

The following steps are still not automated in [user_dedicated_admin_permissions.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/user_dedicated_admin_permissions.go). Once automated, the manual steps should be removed from this test case.

### View only permissions for RHMI Custom Resource

JIRA: [INTLY-7748](https://issues.redhat.com/browse/INTLY-7748)

1. Go to the **redhat-rhmi-operator** namespace in the OpenShift console
2. Go to **Home** > **Search**
3. Select **RHMI** in the dropdown
   > Verify that you are able to view the `rhmi` custom resource in the RHMI operator namespace
4. Create a new RHMI custom resource
   > Verify that you cannot create a new RHMI custom resource
5. Click on the **rhmi** custom resource
   > Verify that you are able to view the RHMI custom resource details and YAML file
6. Change a value in the RHMI custom resource `spec` field
7. Click **Update**
   > Verify that the RHMI custom resource was not updated
8. Click on **Actions** > **Delete RHMI**
   > Verify that the RHMI custom resource was not deleted
