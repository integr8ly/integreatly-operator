---
estimate: 15m
---

# F04 - Verify AWS s3 resources exist and are in expected state

## Description

### AWS Resources

#### S3 Bucket:

- 1 for 3scale (name: rhmibyocjlkcwredhat**rhmioperatorthre**-mpm4)
- 1 for backup (name: rhmibyocjlkcwredhat**rhmioperatorback**-6ibv)

## Prerequisites

Need to login to AWS with the same user used to create the cluster.

## Steps

1. Open and login to the [AWS Console](console.aws.amazon.com)
2. Go to the **S3** Service and resources are present in AWS
   > All [S3 resources](#s3-bucket) should be present
   >
   > For each S3 resource:
   >
   > - The Access column should report `Bucket and objects not public`
