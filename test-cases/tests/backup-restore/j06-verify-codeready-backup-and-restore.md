---
components:
  - product-codeready
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.8.0
estimate: 1h
tags:
  - destructive
---

# J06 - Verify CodeReady Backup and Restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Create Workspace as `test-user`
2. On creation of workspace using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace verify workspace is in DB

```
# password and host retrieved from codeready-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from workspace;
```

3. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/codeready_backup.md#codeready-postgres)
4. Verify workspace is in postgres, follow `Step 2.`
5. Verify you can log into workspace

### PV

1. Create Workspace as `test-user`
2. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/codeready_backup.md#codeready-workspace-pv)
3. Verify workspace exists and can be used
