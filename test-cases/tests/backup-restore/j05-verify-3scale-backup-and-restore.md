---
components:
  - product-3scale
environments:
  - osd-post-upgrade
estimate: 3h
targets:
  - 2.7.0
---

# J05 - Verify 3scale backup and restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Verify data exists in postgres using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace

```
# password and host retrieved from threescale-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from plans;
$ select * from accounts;
```

3. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#postgres)
4. Verify the same data exist in postgres follow `Step 2.`

### Redis

#### Backend Redis

1. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#backend-redis)

#### System Redis

1. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#system-redis)

### System App

1. Once all pods are up and running follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#system-app) and verify 3scale service is working
