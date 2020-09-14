---
components:
  - product-sso
environments:
  - osd-post-upgrade
estimate: 1h
tags:
  - destructive
targets:
  - 2.8.0
---

# J08 - Verify User SSO Backup and Restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Verify Clients and Realms exist in postgres

   Create a throwaway Postgres instance to access the SSO database

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

   Open a terminal in the Pod created for the throwaway Postgres and run the following

   ```sh
   # password and host retrieved from rhssouser-postgres-rhmi secret in redhat-rhmi-operator, psql will prompt for password
   psql --host=<<db host> --port=5432 --username=postgres --password --dbname=postgres
   $ select * from client;
   $ select * from realm;
   ```

   Once verified. Delete the throwaway Postgres

   ```sh
   oc delete -n redhat-rhmi-operator postgres/throw-away-postgres
   ```

2. Run the backup and restore script

   ```sh
   cd test/scripts/backup-restore
   ./j08-verify-user-sso-backup-and-restore.sh | tee test-output.txt
   ```

3. Wait for the script to finish without errors
4. Verify in the `test-output.txt` log that the test finished successfully.
