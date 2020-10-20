---
products:
  - name: rhmi
    environments:
      - osd-fresh-install
    targets:
      - 2.5.0
      - 2.6.0
estimate: 15m
---

# G15 - Verify solution explorer customer settings backup

## Steps

1. Login as dedicated-admin user to the solution explorer
2. Select **?** symbol in the top right hand corner and click on about
   > Verify the Solution Explorer version and confirm that it matches the version obtained via cli:
   >
   > ```
   > oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq -r '.status.stages."solution-explorer".products."solution-explorer".version'
   > ```
3. Close About and select the settings cog in the top right hand corner.
4. From the **Daily Backups** drop down menu select a time to do the backup.
5. From the **Weekly maintenance window** drop down menus, select a day and time for maintenance.
6. Login to the cli as cluster-admin user `oc login ...`
7. Retrieve the **rhmi-config** CR
   ```
   oc get rhmiconfig rhmi-config -n redhat-rhmi-operator -o yaml
   ```
   > Verify that the changes you made in previous steps have been applied correctly to the CR
   >
   > Example:
   >
   > ```
   > spec:
   >   backup:
   >     applyOn: '07:00'
   >   maintenance:
   >     applyFrom: 'Tue 04:00'
   >   upgrade:
   >     notBeforeDays: 7
   >     waitForMaintenance: true
   > ```
