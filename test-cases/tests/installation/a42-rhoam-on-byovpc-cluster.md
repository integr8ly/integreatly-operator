---
products:
  - name: rhoam
    environments:
      - external
estimate: 4h
tags:
  - per-release
---

# A42 - RHOAM on BYOVPC cluster

## Description

Verify RHOAM installation on BYOVPC cluster (OSD cluster using custom AWS VPC)

## Prerequisites

- AWS account (credentials) for deploying custom VPC and OSD cluster
- OCM account (staging environment)

## Steps

1. Log in to AWS (https://<AWS-ACCOUNT-ID>.signin.aws.amazon.com/console)
2. Select region **us-east-1** (top right corner)
3. Search for VPC and select it
4. In left panel select **Elastic IPs** and in top right corner click on **Allocate Elastic IP address**
5. Click on **Allocate**, then note down the Allocation ID of created IP address
6. In left panel select **VPC dashboard** and click on **Launch VPC Wizard**
7. Select **VPC with Public and Private Subnets** and click on **Select**
8. Fill in following fields:

```
IPv4 CIDR block: 10.11.128.0/23
VPC name: byovpc

Public subnet's IPv4 CIDR: 10.11.128.0/24
Availability Zone: us-east-1a

Private subnet's IPv4 CIDR: 10.11.129.0/24
Availability Zone: us-east-1a

Elastic IP Allocation ID: <select the one you previously created>
```

9. Click on **Create VPC** and wait until it's created
10. Back in the left panel select **Subnets**
11. Note down Subnet IDs of subnets named **Public subnet** and **Private subnet**
12. In your terminal, clone delorean repo (`git clone git@github.com:integr8ly/delorean.git`)
13. Export AWS account environment variables (see details in delorean docs: `docs/ocm/README.md`)
14. Export variables for BYOVPC cluster and generate the cluster.json (template) file

```
export BYOVPC=true
export PRIVATE_SUBNET_ID=<your-private-subnet-id>
export PUBLIC_SUBNET_ID=<your-public-subnet-id>
export AVAILABILITY_ZONES=us-east-1a
export OCM_CLUSTER_REGION=us-east-1
export OCM_CLUSTER_NAME=byovpc-<your-redhat-nick>-<AWS-ACCOUNT-ID>
make ocm/cluster.json
```

15. Double check the configuration file and then create a cluster

```
make ocm/cluster/create
```

16. Once it's created, install RHOAM and run the e2e test suite using [Jenkins pipeline](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow/)
