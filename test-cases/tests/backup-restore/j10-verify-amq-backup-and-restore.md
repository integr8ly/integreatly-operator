---
components:
  - product-amq
environments:
  - osd-post-upgrade
estimate: 2h
tags:
  - destructive
targets:
  - 2.4.0
  - 2.8.0
---

# J10 - Verify AMQ backup and restore

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

### Postgres

1. Follow PostgreSQL [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/amq_online_backup.md#amq-online-backup-and-restoration-rhmi-on-2x) section

### Brokered Queue PV

1. Create some data in order to have something to verify.

   1. Navigate to the AMQ console and log in
   2. Create a Brokered Address Space
   3. Create a Brokered Queue Address in the Address Space

2. Follow Brokered Queue PV [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/amq_online_backup.md#2-brokered-queue-pv) section

3. Verify that the above data items are restored

### AMQ Resources

1. Create some data in order to have something to verify.

   1. From the OpenShift Console
   2. Navigate to the redhat-rhmi-amq-online namespace
   3. Create one or more of the below resources by searching for the type and copying an existing entry. Change the name to something recognisable in the verification step
      - AddressPlan
      - AddressSpacePlan
      - AuthenticationService
      - BrokeredInfraConfig
      - StandardInfraConfig

2. Follow AMQ Resources [sop](https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/amq_online_backup.md#3-amq-resources-backup) section

3. Verify the restoration of the resources created in step 2.
