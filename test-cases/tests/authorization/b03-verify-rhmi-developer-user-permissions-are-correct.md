---
automation:
  - INTLY-7748
environments:
  - osd-post-upgrade
  - osd-fresh-install
estimate: 15m
tags:
  - per-release
---

# B03 - Verify RHMI Developer User Permissions are Correct

**Automated Test**: [user_rhmi_developer_permissions.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/user_rhmi_developer_permissions.go)

## Steps

The following steps are still not automated in [user_rhmi_developer_permissions.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/user_rhmi_developer_permissions.go). Once automated, the manual steps should be removed from this test case.

### Manage Address Spaces and Addresses in AMQ Online

JIRA: [INTLY-5434](https://issues.redhat.com/browse/INTLY-5434)

1. Log into AMQ Online via Solution Explorer as a RHMI developer user (e.g. as a test-userXX)
2. Click **Create Address Space** (If **Create Address Space** button is not visible, click the home link, **Red Hat AMQ**. Top left).
3. Enter the following details in the configuration form:
   - Namespace: Select the namespace with the format `<username>-shar-<uid>`
   - Name: Enter any name
   - Type: Select `Standard`
   - Address Space Plan: Select `Unlimited`
   - Authenticaton Service: `none-authservice`
4. Click **Next**
5. Click **Finish**
   > Verify that you are able to see the address space that you have just created in the AMQ Online dashboard. Once ready, the `Status` field of your address space should be set to `Active` (this may take up to ~3mins).
6. Click on the Address space you've just created
7. Click **Create Address**
8. Enter the following details in the form:
   - Address: `test`
   - Type: Select `queue`
   - Plan: Select `Small Queue`
9. Click **Next**
10. Click **Finish**
    > Verify that you are able to see the address you have just created. Once ready, the `Status` field of your address should be set to `Active` (this may take up to ~3mins).
11. Login as a different RHMI developer user to the AMQ Online Console (e.g. as a test-userXX)
    > Verify that you cannot see any address spaces and addresses created by the previous user

### No Access to RHMI Custom Resource

JIRA: [INTLY-7792](https://issues.redhat.com/browse/INTLY-7792)

1. Navigate to the console & log in as an RHMI developer user (e.g. as a test-userXX)
2. Go to **Home** > **Search**
3. Select **RHMI** from the custom resource dropdown
   > Verify that you are not be able to view any RHMI custom resources
