---
estimate: 1h
require:
  - J03
---

# J06 - Verify CodeReady Backup and Restore

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Postgres

1. Create Workspace as `test-user`
2. On creation of workspace using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace verify workspace is in DB

```
# password and host retrieved from codeready-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from workspaces;
```

3. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/codeready_backup.md#codeready-postgres) to backup the database.
4. Delete the workspace created in `Step 1`.
5. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/codeready_backup.md#codeready-postgres) to restore the database.
6. Verify workspace created in `Step 1` exists.
7. Verify you can log into workspace.

## PV

1. Create Workspace as `test-user`.
2. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/codeready_backup.md#codeready-workspace-pv) to backup the PV.
3. Delete the workspace created in `Step 1`.
4. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/codeready_backup.md#codeready-workspace-pv) restore the PV.
5. Verify workspace exists and can be used.
