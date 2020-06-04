---
estimate: 3h
require:
  - J03
---

# J05 - Verify 3scale backup and restore

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Postgres

1. Login to 3scale as a customer admin and create a `Product`.
2. Verify data exists in postgres using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace

```
# password and host retrieved from threescale-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from plans;
$ select * from accounts;
```

3. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#postgres) to back up the database.
4. Login to 3scale as a customer admin and delete the product created in `Step 1`.
5. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#postgres) to restore the database.
6. Verify the same data exist in postgres follow `Step 2.`

## Redis

### Backend Redis

1. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#backend-redis)

### System Redis

1. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#system-redis)

### System App

1. Once all pods are up and running follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#system-app) and verify 3scale service is working
