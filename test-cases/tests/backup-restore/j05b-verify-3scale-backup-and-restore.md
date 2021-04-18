---
components:
  - product-3scale
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.5.0
estimate: 3h
tags:
  - destructive
---

# J05B - Verify 3scale backup and restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Login via `oc` as **kubeadmin**

2. Verify data exist in postgres.

Create a throwaway Postgres instance to access the RHSSO Postgres instance

```sh
cat << EOF | oc create -f - -n redhat-rhoam-operator
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
# Wait until the throwaway Postgres instance is running
oc get pods -n redhat-rhoam-operator | grep throw-away | awk '{print $3}'
# oc rsh to the pod
oc rsh -n redhat-rhoam-operator $(oc get pods -n redhat-rhoam-operator | grep throw-away | awk '{print $1}')
# password and host retrieved from threescale-postgres-rhoam secret in redhat-rhoam-operator, psql will prompt for password
psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
$ select * from plans;
$ select * from accounts;
```

Once verified, delete the throwaway Postgres

```sh
oc delete -n redhat-rhoam-operator postgres/throw-away-postgres
```

3. Run the backup and restore script

```sh
cd test/scripts/backup-restore
NS_PREFIX=redhat-rhoam ./j05-verify-3scale-postgres-backup-and-restore.sh | tee test-output.txt
```

4. Wait for the script to finish without errors
5. Verify in the `test-output.txt` log that the test finished successfully.

### Redis

#### Backend Redis

1. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#backend-redis)

#### System Redis

1. Follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#system-redis)

### System App

1. Once all pods are up and running follow [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/3scale_backup.md#system-app) and verify 3scale service is working
