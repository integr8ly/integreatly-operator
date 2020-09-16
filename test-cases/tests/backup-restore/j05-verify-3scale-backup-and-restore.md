---
components:
  - product-3scale
environments:
  - osd-post-upgrade
estimate: 1h
tags:
  - destructive
targets:
  - 2.7.0
---

# J05 - Verify 3scale backup and restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Pre-requisites

1. Create a throw-away Postgres instance

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

- Run the following commands in throwaway Postgres pod and verify that data in 3Scale Postgres exists:

```
# password and host retrieved from threescale-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from plans;
$ select * from accounts;
```

2. Delete Postgres throw-away pod:

```sh
oc delete -n redhat-rhmi-operator postgres/throw-away-postgres
```

3. Navigate to https://3scale-admin.[Your_admin_domain].devshift.org/p/admin/onboarding/wizard/intro and follow the steps outlined in the wizard to create a 3Scale service. You can get your admin domain from `Routes` under `redhat-rhmi-3scale` namespace.

### Backup and restore

1. Run the backup and restore script

```sh
cd test/scripts/backup-restore
./j05-verify-3scale-backup-and-restore.sh | tee test-output.txt
```

2. Wait for the script to finish without errors
3. Verify in the `test-output.txt` log that the test finished successfully.
4. Verify that 3Scale service is still functional by:

- Log in to 3Scale
- On the dashboard page select `Integration` of your `API` product
- Copy staging API Cast `cURL` link and run it in a terminal
