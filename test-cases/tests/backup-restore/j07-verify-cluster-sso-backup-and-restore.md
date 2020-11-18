---
components:
  - product-sso
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.6.0
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
estimate: 1h
tags:
  - destructive
targets:
  - 2.6.0
  - 2.7.1
---

# J07 - Verify Cluster SSO Backup and Restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Login via `oc` as **kubeadmin**

2. Verify Clients and Realms exist in postgres.

Create a throwaway Postgres instance to access the RHSSO Postgres instance

```sh
cat << EOF | oc create -f - -n redhat-rhmi-operator
  apiVersion: integreatly.org/v1alpha1
  kind: Postgres
  metadata:
    name: throw-away-postgres
    labels:
      productName: productName
  spec:
    secretRef:
      name: throw-away-postgres-sec
    tier: development
    type: workshop
EOF
```

```
# password and host retrieved from rhsso-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgresuser --password --dbname=rhsso-postgres-rhmi
$ select * from client;
$ select * from realm;
```

Once verified. Delete the throwaway Postgres

```sh
oc delete -n redhat-rhmi-operator postgres/throw-away-postgres
```

3. Run the backup and restore script

```sh
cd test/scripts/backup-restore
./j07-verify-rhsso-backup-and-restore.sh | tee test-output.txt
```

4. Wait for the script to finish without errors
5. Verify in the `test-output.txt` log that the test finished successfully.
