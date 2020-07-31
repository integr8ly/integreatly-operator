---
components:
  - product-fuse
environments:
  - osd-post-upgrade
estimate: 1h
tags:
  - destructive
targets:
  - 2.8.0
---

# J09 - Verify Fuse Backup and Restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Create connection in Fuse to Postgres DB, use sampleDB as example
2. Create basic integreation to new connection
   1. Simple SQL query and log
3. Using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace, verify data in postgres

```
# password and host retrieved from fuse-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ \dt
```

4. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/fuse_online_backup_and_restore.md#fuse-online-backup-and-restoration)
5. Verify integreation is restored
