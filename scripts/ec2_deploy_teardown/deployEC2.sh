#!/bin/bash

echo "Creating key-pair"
aws ec2 create-key-pair --key-name pipelineKey --query 'KeyMaterial' --output text --region ${CLUSTER_REGION} > pipelineKey.pem && Keyfile=true

echo "Getting VPC Details"
VPC=`aws ec2 describe-vpcs --region ${CLUSTER_REGION} --filters "Name=tag:Name,Values=${clusterID}*" --query "Vpcs[0].VpcId" --output text` 
for (( i=0; i<5; i++ )) do
    if [[ $VPC =~ "vpc-" ]]; then
      echo "VPC "${VPC}" is available"
      break
    fi
    sleep 5
done

echo "Getting subnet details"
SUBNET=`aws ec2 describe-subnets --region ${CLUSTER_REGION} --filters "Name=vpc-id,Values=$VPC" "Name=tag:Name,Values=*public*" --query Subnets[0].SubnetId --output text`
for (( i=0; i<5; i++ )) do
    if [[ $SUBNET =~ "subnet-" ]]; then
      echo "SUBNET "${SUBNET}" is available"
      break
      else 
      echo $SUBNET
    fi
    sleep 5
done

echo "Creating security group"
SECGRP=`aws ec2 create-security-group --group-name pipeline-sg --description "pipeline security group" --vpc-id $VPC --region ${CLUSTER_REGION} --output text`
for (( i=0; i<5; i++ )) do
    if [[ $SECGRP =~ "sg-" ]]; then
      echo "SECGRP "${SECGRP}" is created"
      break
    fi
    sleep 5
done

echo "Authorising security group"
aws ec2 authorize-security-group-ingress --group-id $SECGRP --protocol tcp --port 22 --cidr 0.0.0.0/0 --region ${CLUSTER_REGION}
sleep 5

echo "Getting image"
AMI=`aws ec2 describe-images --filters "Name=is-public,Values=true" "Name=description,Values=Provided by Red Hat*" "Name=name,Values=${IMAGE}" --region ${CLUSTER_REGION} --output text --query 'Images[0].ImageId'`
for (( i=0; i<5; i++ )) do
    if [[ $AMI =~ "ami-" ]]; then
      echo "AMI "${AMI}" is available"
      break
    fi
    sleep 5
done

echo "Starting EC2 micro-t2 instance"
EC2=`aws ec2 run-instances --image-id $AMI --count 1 --instance-type t2.micro --key-name pipelineKey --security-group-ids $SECGRP --subnet-id $SUBNET --region ${CLUSTER_REGION} --query 'Instances[0].InstanceId' --output text`
for (( i=0; i<60; i++ )) do
    commandResult=$(aws ec2 describe-instance-status --region $CLUSTER_REGION --instance-ids $EC2 --query 'InstanceStatuses[0].InstanceState.Name' --output text)
    if [[ $commandResult == "running" ]]; then
      echo "ec2 "$EC2" instance is running"
      break
    fi
    sleep 10
done

echo "Allocating address"
ELIP=`aws ec2 allocate-address --domain vpc --region ${CLUSTER_REGION} --query 'AllocationId' --output text`
for (( i=0; i<5; i++ )) do
    if [[ $ELIP =~ "eipalloc-" ]]; then
      echo "Address "${ELIP}" is allocated"
      break
    fi
    sleep 5
done

echo "Associating address"
ASSOC=`aws ec2 associate-address --allocation-id $ELIP --instance-id $EC2 --region ${CLUSTER_REGION}  --output text --query 'AssociationId'`
for (( i=0; i<5; i++ )) do
    if [[ $ASSOC =~ "eipassoc-" ]]; then
      echo "Address "${ASSOC}" is associated"
      break
    fi
    sleep 5
done

echo "Getting public ip"
PUBIP=`aws ec2 describe-instances --filters "Name=instance-id,Values=$EC2" --query 'Reservations[].Instances[].PublicDnsName' --region ${CLUSTER_REGION} --output text`
for (( i=0; i<5; i++ )) do
    if [[ $PUBIP =~ "compute-1.amazonaws.com" ]]; then
      echo "Public IP "${PUBIP}" is available"
      break
    fi
    sleep 5
done

INSTANCE=`aws ec2 describe-instances --instance-ids $EC2 --region ${CLUSTER_REGION} --output json`
echo ${INSTANCE}
echo ""
echo "If working locally, you will need to create the pipelineKey.pem and paste in the following contents before running the following commands."
echo ""
cat pipelineKey.pem 
echo ""
echo "chmod 700 pipelineKey.pem"
echo "ssh -i pipelineKey.pem ec2-user@$PUBIP"

