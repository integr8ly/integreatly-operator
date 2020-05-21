---
estimate: 1h
---

# J01 - Verify that all in-cluster backups cron jobs succeed

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

Acceptance Criteria:

1. All cron jobs should succeed
   1. enmasse-pv-backup
   2. codeready-pv-backup
   3. enmasse-postgres-backup
2. All backups files should be stored in the S3 Bucket
   1. CodeReady PV
   2. EnMasse PV

Steps:

Manually Triggering the Cron Jobs

1. Enmasse-pv-backup

   1. Create AMQ-online address to generate data

      `The AMQ-online console can only be accessed via the Solution Explorer, signed in as test_user**`

      1. select AMQ-online console from solution explorer landing page

      2. select Address from left hand menu

      3. put a name in the name field of the pop up and select queue

      4. press Next

      5. select small queue

      6. press Next

      7. press Create

      8. wait for address creation to complete

   2. Trigger enmasse-pv-backup Cron Job

      `The cron job can only be triggered from the openshift console signed in as kubeadmin`

      1. go to Projects/redhat-rhmi-amq-online namespace

      2. go to Workloads/Cron Jobs

      3. select enmasse-pv-backup

      4. select YAML

      5. in YAML update spec/schedule to trigger cron job and save.

         Example: As the cluster is deployed to AWS Ireland and the time is currently 4pm Irish and you want to run the job in 2 minutes, then the schedule should be 2 16 \* \* \*.

         If you are in a different timezone you will need to adjust for the timezone where the cluster is deployed. If it is 5pm in Brno the schedule should still be 2 16 \* \* \* as the cluster is deployed to aws Ireland.

      6. save the YAML

      7. select Events tab

      8. wait until you see the job has been completed

2. Codeready-pv-backup

   1. Create codeready workspace to generate data

      `The CodeReady console can only be accessed via the Solution Explorer, signed in as test_user**`

      1. select codeready console from solution explorer landing page

      2. select Create Workspace

      3. From Projects select kitchensink-example

      4. select CREATE & OPEN

      5. wait for the workspace to complete and open.

   2. Trigger codeready-pv-backup Cron Job

      `The cron job can only be triggered from the openshift console signed in as kubeadmin`

      1. Projects/redhat-rhmi-codeready-workspaces namespace

      2. go to Workloads/Cron Jobs

      3. select codeready-pv-backup

      4. select YAML

      5. in YAML update spec/schedule to trigger cron job and save.

         Example: As the cluster is deployed to AWS Ireland and the time is currently 4pm Irish and you want to run the job in 2 minutes, then the schedule should be 2 16 \* \* \*.

         If you are in a different timezone you will need to adjust for the timezone where the cluster is deployed. If it is 5pm in Brno the schedule should still be 2 16 \* \* \* as the cluster is deployed to aws Ireland.

      6. save the YAML

      7. select Events tab

      8. wait until you see the job has been completed

3. Enmasse-postgres-backup

   1. Trigger enmasse-postgres-backup Cron Job

      `The cron job can only be triggered from the openshift console signed in as kubeadmin`

      1. go to Projects/redhat-rhmi-amq-online namespace

      2. go to Workloads/Cron Jobs

      3. select enmasse-postgres-backup

      4. select YAML

      5. in YAML update spec/schedule to trigger cron job and save.

         Example: As the cluster is deployed to AWS Ireland and the time is currently 4pm Irish and you want to run the job in 2 minutes, then the schedule should be 2 16 \* \* \*.

         If you are in a different timezone you will need to adjust for the timezone where the cluster is deployed. If it is 5pm in Brno the schedule should still be 2 16 \* \* \* as the cluster is deployed to aws Ireland.

      6. save the YAML

      7. select Events tab

      8. wait until you see the job has been completed

Verifying the Cron Jobs Have Been Run

    You will need the credentials for the aws account where jobs are backed up. Request these from a cloud-services-qe member if needed.

1. Check the S3 bucket name being used by the cronjob
   1. rhmi-redhat-<product> -> Workloads -> Secrets -> backups-s3-credentials -> AWS_S3_BUCKET_NAME
2. Login to the AWS S3 console: https://s3.console.aws.amazon.com/s3/home?region=eu-west-1
3. Select the bucket
4. Select the backups folder
5. Select the amqonline folder
   1. select enmasse_pv
   2. navigate through the folder in descending date order checking you have the correct date and time for the backup you are looking for
   3. There should be a backup file for enmasse - example: \*enmasse-pv-data-10_58_15.tar.gz
6. Select the codeready folder
   1. select codeready_pv
   2. navigate through the folder in descending date order checking you have the correct date and time for the backup you are looking for
   3. There should be a backup file for codeready - example: _codeready-pv-data-09_18_18.tar.gz_
7. Select the amqonline folder
   1. select postgres
   2. navigate through the folder in descending date order checking you have the correct date and time for the backup you are looking for
   3. There should be a backup file for enmasse - example: \*standard-authservice-postgresql-09_18_18.tar.gz
