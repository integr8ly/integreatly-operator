#!/bin/bash

echo "Obtaining security group"
SECGRP=`aws ec2 describe-security-groups --region ${CLUSTER_REGION} --filters Name=group-name,Values=pipeline-sg --query SecurityGroups[0].GroupId --output text`
for (( i=0; i<5; i++ )) do
    if [[ $SECGRP =~ "sg-" ]]; then
      echo "SECGRP "${SECGRP}" identified"
      break
    fi
    sleep 5
done

echo "Obtaining EC2 micro-t2 instance"
EC2=`aws ec2 describe-instances  --region ${CLUSTER_REGION} --filters Name=key-name,Values=pipelineKey Name=instance-state-name,Values=running --query Reservations[0].Instances[0].InstanceId --output text`
for (( i=0; i<5; i++ )) do
    if [[ $EC2 =~ "i-" ]]; then
      echo "EC2 "${EC2}" identified"
      break
    fi
    sleep 5
done

echo "Obtaining address allocation details"
ELIP=`aws ec2 describe-addresses --filters Name=instance-id,Values=$EC2 --region ${CLUSTER_REGION} --query Addresses[0].AllocationId --output text`
for (( i=0; i<5; i++ )) do
    if [[ $ELIP =~ "eipalloc-" ]]; then
      echo "Address "${ELIP}" identified"
      break
    fi
    sleep 5
done

echo "Obtaining address association details"
ASSOC=`aws ec2 describe-addresses --filters Name=instance-id,Values=$EC2 --region ${CLUSTER_REGION} --query Addresses[0].AssociationId --output text`
for (( i=0; i<5; i++ )) do
    if [[ $ASSOC =~ "eipassoc-" ]]; then
      echo "Address "${ASSOC}" identified"
      break
    fi
    sleep 5
done

echo "Disassociating address "$ASSOC
aws ec2 disassociate-address --association-id $ASSOC --region $CLUSTER_REGION

echo "Releasing address "$ELIP
aws ec2 release-address --allocation-id $ELIP --region $CLUSTER_REGION

echo "Terminatinng EC2 instance "$EC2
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

echo "Deleting key-pair pipelineKey from aws"
aws ec2 delete-key-pair --key-name pipelineKey --region $CLUSTER_REGION


