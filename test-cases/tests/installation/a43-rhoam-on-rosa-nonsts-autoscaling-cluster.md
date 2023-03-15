---
products:
  - name: rhoam
    environments:
      - external
    targets:
      - 1.27.0
      - 1.30.0
      - 1.33.0
estimate: 4h
tags:
  - automated
---

# A43 - RHOAM on ROSA non-STS Autoscaling Cluster

## Description

Verify RHOAM installation on Autoscaling ROSA non-STS

## Prerequisites

- AWS account with ROSA enabled - the QE AWS account used for ROSA nightly pipeline can be used
- OCM access to where cluster was created - OCM token from parent epic can be used

## Steps

1. Provision a ROSA non-STS autoscaling cluster

Use [https://github.com/integr8ly/delorean/pull/284] if not merged yet. `make ocm/rosa/cluster/create` should do the trick. Make sure 4 nodes is minimum and 6 nodes maximum and STS is disabled via following env vars:

- ENABLE_AUTOSCALING=true
- MIN_REPLICAS=4
- MAX_REPLICAS=6
- STS_ENABLED=false

Under the hood this should be called:
`rosa create cluster --cluster-name $CLUSTER_NAME --region=$AWS_REGION --compute-machine-type=$MACHINE_TYPE -y --enable-autoscaling --min-replicas $MIN_REPLICAS --max-replicas $MAX_REPLICAS`

2. Install current GA version of RHOAM on the cluster using 1M quota

E.g.: If testing RHOAM v1.28.0 rc1 then install RHOAM v1.27.0. Use rosa cli addon installation - ROSA pipeline can also be used. Example: `rosa install addon managed-api-service --cluster=${clusterName} --addon-managed-api-service=10 --addon-resource-required=true --cidr-range=10.1.0.0/26`
Then patch the RHMI rhoam CR:
`oc patch rhmi rhoam -n redhat-rhoam-operator --type=merge -p '{"spec":{"useClusterStorage": "false" }}'`

3. Wait for installation to complete

Once complete, do a quick manual check or run `Setup IDP` and `Sanity Check` steps via pipeline.

4. Deploy workload web app
5. Wait for desired RHOAM version (usually RC1) to make it into the OCM stage. Upgrade should start automatically
6. Watch the upgrade and once finished review workload web app Grafana dashboard for downtimes
7. Change quota to 50M via OCM

Wait for `toVersion` to appear in RHMI rhoam CR and disappear. Once disappeared check the workload web app Grafana dashboard for downtimes. Also check the cluster, it should autoscale from four worker nodes to five.
