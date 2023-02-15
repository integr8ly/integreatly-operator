#!/bin/bash

set -e

echo "Getting vpc network details..."
VPC_NAME=`$GCLOUD compute networks list --filter="name=$CLUSTER_INFRA_NAME-network" --format="value(name)"`
if [ -z "$VPC_NAME" ]; then
  echo "VPC network $CLUSTER_INFRA_NAME-network not found"
  exit 1
fi
echo "VPC network $VPC_NAME is available"

echo "Getting worker subnet details..."
WORKER_SUBNET_NAME=`$GCLOUD compute networks subnets list --network $VPC_NAME --filter="name~^.*worker.*$" --format="value(name)"`
if [ -z "$WORKER_SUBNET_NAME" ]; then
  echo "Worker subnet for network $VPC_NAME not found"
  exit 1
fi
echo "Worker subnet $WORKER_SUBNET_NAME for network $VPC_NAME is available"

echo "Getting compute vm image..."
VM_IMAGE_NAME=`$GCLOUD compute images list --filter="name=$VM_IMAGE" --format="value(name)"`
if [ -z "$VM_IMAGE_NAME" ]; then
  echo "Compute vm image $VM_IMAGE not found"
  exit 1
fi
echo "Compute vm image $VM_IMAGE is available"
