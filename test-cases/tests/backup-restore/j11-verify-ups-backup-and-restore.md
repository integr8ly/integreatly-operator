---
estimate: 1h
targets: []
require:
  - J03
---

# J11 - Verify UPS backup and restore

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Postgres

1. Create a dummy application and an android variant through UPS console
2. On creation of dummy application and android variant, using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace verify data in DB

```
# password and host retrieved from ups-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from android_variants;
```

3. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/ups_backup_and_restore.md#unified-push-server-ups-backup-and-restoration)
4. Verify data is in Postgres, repeat `Step 2.`
