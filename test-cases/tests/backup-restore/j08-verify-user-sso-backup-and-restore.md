---
components:
  - product-sso
environments:
  - osd-post-upgrade
estimate: 1h
tags:
  - destructive
targets:
  - 2.7.0
---

# J08 - Verify User SSO Backup and Restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Verify Clients and Realms exist in postgres using the terminal in the `standard-auth` pod in the `redhat-rhmi-operator` namespace

```
# password and host retrieved from rhssouser-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from clients;
$ select * from realms;
```

3. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/user_sso_backup.md)
4. Verify the same clients and realms exist in postgres follow `Step 2.`
