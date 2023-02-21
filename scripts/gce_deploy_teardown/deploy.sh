#!/bin/bash

set -e

draw_horizontal_line () {
  echo -e '\n'
  printf '%.s─' $(seq 1 $(tput cols))
  echo -e '\n'
}

draw_horizontal_line
echo "Getting vpc network details..."
VPC_NETWORK=$($GCLOUD compute networks list --filter="name=$VPC_NAME" --format="value(name)")
if [ -z "$VPC_NETWORK" ]; then
  echo "VPC network $VPC_NETWORK not found"
  exit 1
fi
echo "VPC network $VPC_NETWORK is available"

draw_horizontal_line
echo "Getting worker subnet details..."
WORKER_SUBNET=$($GCLOUD compute networks subnets list --network $VPC_NETWORK --filter="name~^.*worker.*$" --format="value(name)")
if [ -z "$WORKER_SUBNET" ]; then
  echo "Worker subnet for vpc network $VPC_NETWORK not found"
  exit 1
fi
echo "Worker subnet $WORKER_SUBNET for network $VPC_NETWORK is available"

draw_horizontal_line
echo "Getting compute vm image..."
VM_IMAGE=$($GCLOUD compute images list --filter="name=$VM_IMAGE_NAME" --format="value(name)")
if [ -z "$VM_IMAGE" ]; then
  echo "Compute vm image $VM_IMAGE not found"
  exit 1
fi
echo "Compute vm image $VM_IMAGE is available"

draw_horizontal_line
echo "Creating vm instance $VM_NAME..."
PROJECT=$($GCLOUD config get-value project)
$GCLOUD compute instances create $VM_NAME --zone=$VM_ZONE --subnet=$WORKER_SUBNET --machine-type=$VM_MACHINE_TYPE --tags=$VM_NAME --metadata=block-project-ssh-keys=true --create-disk=auto-delete=yes,boot=yes,device-name=rhel-8-1,image=projects/rhel-cloud/global/images/$VM_IMAGE,mode=rw,size=50GB,type=projects/$PROJECT/zones/us-central1-a/diskTypes/pd-balanced

draw_horizontal_line
echo "Adding SSH key to vm instance $VM_NAME..."
$GCLOUD compute instances add-metadata $VM_NAME --zone=$VM_ZONE --metadata=ssh-keys="vmuser:$PUBLIC_SSH_KEY"

draw_horizontal_line
JENKINS_IP=$(curl ifconfig.me)
echo "Creating firewall rule to allow $JENKINS_IP, $FULL_ACCESS_IP, 127.0.0.1 access to all ports for ingress connections on vm instance $VM_NAME..."
$GCLOUD compute firewall-rules create $VM_NAME-firewall-rule --direction=INGRESS --network=$VPC_NETWORK --action=ALLOW --rules=all --source-ranges=$JENKINS_IP,$FULL_ACCESS_IP,127.0.0.1 --target-tags=$VM_NAME

draw_horizontal_line
echo "Information about the newly created vm instance $VM_NAME:"
$GCLOUD compute instances describe $VM_NAME --zone=$VM_ZONE

draw_horizontal_line
echo "To SSH into the created instance using your private key (it has to be from the same key pair as the public key passed via environment variable 'SSH_PUBLIC_KEY'):"
echo "gcloud compute ssh $VM_NAME --zone $VM_ZONE --project $PROJECT --ssh-key-file <private key file path>"
