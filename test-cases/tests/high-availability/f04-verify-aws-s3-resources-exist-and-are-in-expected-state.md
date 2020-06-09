---
estimate: 15m
---

# F04 - Verify AWS s3 resources exist and are in expected state

## Description

## Prerequisites

- Make sure you have the [AWS CLI](https://aws.amazon.com/cli/) installed, and configured correctly.
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
   ```
   RESOURCETAGMAPPINGLIST  arn:aws:s3:::addonflow1154pgnwredhatrhmioperator-misj
   TAGS    integreatly.org/clusterID       addon-flow-115-4pgnw
   TAGS    integreatly.org/product-name    cloud-resources
   TAGS    integreatly.org/resource-name   backups-blobstorage-rhmi
   TAGS    integreatly.org/resource-type   managed
   RESOURCETAGMAPPINGLIST  arn:aws:s3:::addonflow1154pgnwredhatrhmioperator-tdhs
   TAGS    integreatly.org/clusterID       addon-flow-115-4pgnw
   TAGS    integreatly.org/product-name    3scale
   TAGS    integreatly.org/resource-name   threescale-blobstorage-rhmi
   TAGS    integreatly.org/resource-type   managed
   ```
