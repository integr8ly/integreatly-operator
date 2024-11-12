---
components:
  - product-sso
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.3.0
      - 1.6.0
      - 1.9.0
      - 1.12.0
      - 1.15.0
      - 1.20.0
      - 1.23.0
      - 1.26.0
      - 1.29.0
      - 1.32.0
      - 1.35.0
      - 1.38.0
      - 1.41.0
estimate: 1h
tags:
  - destructive
---

# J08B - Verify User SSO Backup and Restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.
In case the current cluster is not available for some reason, you can perform this test case on the "fresh install" cluster

## Steps

### Postgres

1. Login via `oc` as **kubeadmin**

2. Verify Clients and Realms exist in postgres.

Create a throwaway Postgres instance to access the User SSO Postgres instance

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
# Get postgres db host and password to use it later for logging into postgres db
oc get secret rhssouser-postgres-rhoam -n redhat-rhoam-operator -o json | jq -r .data.host | base64 --decode
oc get secret rhssouser-postgres-rhoam -n redhat-rhoam-operator -o json | jq -r .data.password | base64 --decode
# Wait until the throwaway Postgres instance is running
oc get pods -n redhat-rhoam-operator | grep throw-away | awk '{print $3}'
# oc rsh to the pod
oc rsh -n redhat-rhoam-operator $(oc get pods -n redhat-rhoam-operator | grep throw-away | awk '{print $1}')
# Log in using host and password retrieved with commands above. psql will prompt for password
psql --host=<postgres_db_host> --port=5432 --username=postgres --password --dbname=postgres
select * from client;
select * from realm;
```

Once verified. Delete the throwaway Postgres

```sh
oc delete -n redhat-rhoam-operator postgres/throw-away-postgres
```

3. **Non-STS** - Run the backup and restore script

```sh
cd test/scripts/backup-restore
NS_PREFIX=redhat-rhoam ./j08-verify-user-sso-backup-and-restore.sh | tee test-output.txt
```

4. **STS** - Reach out to QE for the `osdCcsAdmin` credentials in order to run the backup and restore script

```sh
cd test/scripts/backup-restore
AWS_ACCESS_KEY_ID=<aws_access_key_id> AWS_SECRET_ACCESS_KEY=<aws_secret_access_key> NS_PREFIX=redhat-rhoam ./j08-verify-user-sso-backup-and-restore.sh | tee test-output.txt
```

5. Wait for the script to finish without errors
6. Verify in the `test-output.txt` log that the test finished successfully.

**Note**
Sometimes there could be a difference between the DB dump files, caused by a changed order of lines in these files. That is not considered to be an issue. See [MGDAPI-2380](https://issues.redhat.com/browse/MGDAPI-2380) for more details. Acceptable difference also is for 'salt' and 'value' values in 'secret_data' column and a value in 'created_date' column in public.credentials table.
