---
automation:
  - INTLY-6654
components:
  - product-3scale
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.1.0
      - 0.2.0
estimate: 15m
tags:
  - automated
---

# F04 - Verify AWS s3 resources exist and are in expected state

## Description

## Prerequisites

- Make sure you have the [AWS CLI](https://aws.amazon.com/cli/) installed, and configured correctly.
  - Reach out to QE to get credentials for the web login in AWS.
  - Once you have logged in, navigate to the user name `tester-XXXX-XXXX-XXXX` on the top right (next to `Global` and `Support` menus).
  - In the dropdown select and navigate to `My Security Credentials`.
  - In the `AWS IAM credentials` tab, create a new access key, note the `Access Key ID` and `Secret Access Key` that will be returned, use these in your CLI to configure aws with `aws configure`.
- Make sure you have `oc` installed and logged into as admin user to the testing cluster.

## Steps

1. Run the following command:
   ```
   aws resourcegroupstaggingapi get-resources \
      --region=$(oc get infrastructure cluster -o jsonpath='{.status.platformStatus.aws.region}') \
      --tag-filters Key=integreatly.org/clusterID,Values=$(oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}') \
      --resource-type-filters s3
   ```
2. You should see 2 buckets listed when the command is finished running. You should also be able to see the tags for each of the bucket. One of these tags should be `integreatly.org/product-name`. One bucket should have value `cloud-resources` for this tag, and the other should have `3scale` for this tag. The following is an example output:

   ```json
   {
     "ResourceTagMappingList": [
       {
         "ResourceARN": "arn:aws:s3:::addonflow2576sjdqredhatrhmioperator-ptkh",
         "Tags": [
           {
             "Key": "integreatly.org/clusterID",
             "Value": "addon-flow-257-6sjdq"
           },
           {
             "Key": "integreatly.org/product-name",
             "Value": "3scale"
           },
           {
             "Key": "integreatly.org/resource-name",
             "Value": "threescale-blobstorage-rhmi"
           },
           {
             "Key": "integreatly.org/resource-type",
             "Value": "managed"
           }
         ]
       },
       {
         "ResourceARN": "arn:aws:s3:::addonflow2576sjdqredhatrhmioperator-hy2s",
         "Tags": [
           {
             "Key": "integreatly.org/clusterID",
             "Value": "addon-flow-257-6sjdq"
           },
           {
             "Key": "integreatly.org/product-name",
             "Value": "cloud-resources"
           },
           {
             "Key": "integreatly.org/resource-name",
             "Value": "backups-blobstorage-rhmi"
           },
           {
             "Key": "integreatly.org/resource-type",
             "Value": "managed"
           }
         ]
       }
     ]
   }
   ```
