#!/bin/bash

set -ex

echo "Obtaining security group"
SECGRP=`aws ec2 describe-security-groups --region ${CLUSTER_REGION} --filters Name=group-name,Values=${EC2_NAME}-sg --query SecurityGroups[0].GroupId --output text`
if [[ $SECGRP =~ "sg-" ]]; then
  echo "SECGRP "${SECGRP}" identified"
else
  echo "Security group not found!"
  exit 1
fi

echo "Obtaining EC2 instance"
EC2=`aws ec2 describe-instances  --region ${CLUSTER_REGION} --filters Name=key-name,Values=${EC2_NAME}Key Name=instance-state-name,Values=running --query Reservations[0].Instances[0].InstanceId --output text`
if [[ $EC2 =~ "i-" ]]; then
  echo "EC2 "${EC2}" identified"
else
  echo "EC2 instance not found!"
  exit 1
fi

echo "Terminating EC2 instance "$EC2
aws ec2 terminate-instances --instance-ids $EC2 --region $CLUSTER_REGION
for (( i=0; i<60; i++ )) do
    commandResult=$(aws ec2 describe-instances  --region ${CLUSTER_REGION} --instance-ids $EC2 --query Reservations[0].Instances[0].State.Name --output text)
    if [[ $commandResult == "terminated" ]]; then
      echo "ec2 instance "$EC2" has terminated"
      break
    fi
    sleep 10
done

echo "Waiting 30s for security-groups network-interface to be removed"
sleep 30

echo "Deleting security group "$SECGRP
aws ec2 delete-security-group --group-id $SECGRP --region $CLUSTER_REGION

echo "Deleting key-pair ${EC2_NAME}Key from aws"
aws ec2 delete-key-pair --key-name ${EC2_NAME}Key --region $CLUSTER_REGION


