---
estimate: 1h
require:
  - J03
---

# J07 - Verify Cluster SSO Backup and Restore

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Postgres

1. Verify Clients and Realms exist in postgres using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace.

```
# password and host retrieved from rhsso-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from clients;
$ select * from realms;
```

2. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/rhsso_backup.md#rhsso-backup-and-restoration) to backup the database.
3. Delete all Realms (which also includes all Clients).
4. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/rhsso_backup.md#rhsso-backup-and-restoration) to restore the database.
5. Verify the same clients and realms exist in postgres follow `Step 1`.
