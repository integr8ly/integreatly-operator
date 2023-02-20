#!/bin/bash

set -e

draw_horizontal_line () {
  echo -e '\n'
  printf '%.s─' $(seq 1 $(tput cols))
  echo -e '\n'
}

draw_horizontal_line
echo "Getting vpc network details..."
VPC_NAME=$($GCLOUD compute networks list --filter="name=$CLUSTER_INFRA_NAME-network" --format="value(name)")
if [ -z "$VPC_NAME" ]; then
  echo "VPC network $CLUSTER_INFRA_NAME-network not found"
  exit 1
fi
echo "VPC network $VPC_NAME is available"

draw_horizontal_line
echo "Getting worker subnet details..."
WORKER_SUBNET_NAME=$($GCLOUD compute networks subnets list --network $VPC_NAME --filter="name~^.*worker.*$" --format="value(name)")
if [ -z "$WORKER_SUBNET_NAME" ]; then
  echo "Worker subnet for network $VPC_NAME not found"
  exit 1
fi
echo "Worker subnet $WORKER_SUBNET_NAME for network $VPC_NAME is available"

draw_horizontal_line
echo "Getting compute vm image..."
VM_IMAGE_NAME=$($GCLOUD compute images list --filter="name=$VM_IMAGE" --format="value(name)")
if [ -z "$VM_IMAGE_NAME" ]; then
  echo "Compute vm image $VM_IMAGE not found"
  exit 1
fi
echo "Compute vm image $VM_IMAGE is available"

draw_horizontal_line
echo "Creating vm instance $VM_NAME..."
PROJECT=$($GCLOUD config get-value project)
$GCLOUD compute instances create $VM_NAME --zone=$REGION-a --subnet=$WORKER_SUBNET_NAME --machine-type=$VM_MACHINE_TYPE --tags=performance-test-vm --metadata=block-project-ssh-keys=true --create-disk=auto-delete=yes,boot=yes,device-name=rhel-8-1,image=projects/rhel-cloud/global/images/$VM_IMAGE_NAME,mode=rw,size=200,type=projects/$PROJECT/zones/us-central1-a/diskTypes/pd-balanced

draw_horizontal_line
echo "Adding SSH key to vm instance $VM_NAME..."
$GCLOUD compute instances add-metadata performance-test-vm --zone=$REGION-a --metadata="ssh-keys=vmuser:$PUBLIC_SSH_KEY"

draw_horizontal_line
echo "Creating firewall rule to allow $FULL_ACCESS_IP access to all ports for ingress connections on vm instance $VM_NAME..."
$GCLOUD compute firewall-rules create performance-test-firewall-rule --direction=INGRESS --network=$CLUSTER_INFRA_NAME-network --action=ALLOW --rules=all --source-ranges=$FULL_ACCESS_IP/32,127.0.0.1/32 --target-tags=performance-test-vm

draw_horizontal_line
echo "Information about the newly created vm instance $VM_NAME:"
$GCLOUD compute instances describe performance-test-vm --zone=$REGION-a

draw_horizontal_line
echo "To SSH into the created instance using your private key (it has to be from the same key pair as the public key passed via environment variable 'SSH_PUBLIC_KEY'):"
echo "gcloud compute ssh --zone $REGION-a $VM_NAME --project $PROJECT --ssh-key-file <private key file path>"
