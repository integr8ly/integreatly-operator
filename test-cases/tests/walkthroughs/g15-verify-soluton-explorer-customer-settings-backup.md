---
estimate: 15m
targets:
  - 2.6.0
---

# G15 - Verify soluton explorer customer settings backup

Acceptance Criteria:

When logged into solution explorer as a user from the dedicated-admin group for example customer-admin01.

1. Select ? in the top right hand corner to check the Solution Explorer version and confirm that it matches the version obtained via cli: `oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq -r '.status.stages."solution-explorer".products."solution-explorer".version'`
2. Select the settings cog in the top right hand corner, it should be active.
3. From the 'Daily Backups' drop down menu select a time to do the backup.
4. From the 'Weekly maintenance window' drop down menus, select a day and time for maintenance.

When logged into openshift as cluster-admin

1. Goto <openshift-cluster>k8s/ns/redhat-rhmi-operator/integreatly.org~v1alpha1~RHMIConfig/rhmi-config/yaml
2. At the bottom of the screen select 'Reload' button.
3. Check the yaml to ensure the selections you made have been updated to the correct values.

```
##for example
spec:
  backup:
    applyOn: '07:00'
  maintenance:
    applyFrom: 'Tue 04:00'
  upgrade:
    notBeforeDays: 7
    waitForMaintenance: true
```
