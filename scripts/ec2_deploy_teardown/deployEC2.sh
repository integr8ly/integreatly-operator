#!/bin/bash

set -ex

echo "Getting VPC Details"
if [[ ${PRIVATE_LINK_ENABLED} ]]; then
  echo "PRIVATE_LINK_ENABLED, search VPC by stack name"
  VPC=`aws ec2 describe-vpcs --region ${CLUSTER_REGION} --filters "Name=tag:aws:cloudformation:stack-name, Values=${CLOUDFORM_STACK_NAME}" --query "Vpcs[0].VpcId" --output text`
else
  echo "search VPC by ClusterName: ${clusterName}"
  VPC=`aws ec2 describe-vpcs --region ${CLUSTER_REGION} --filters "Name=tag:Name,Values=${clusterName}*" --query "Vpcs[0].VpcId" --output text`
fi

if [[ $VPC =~ "vpc-" ]]; then
  echo "VPC "${VPC}" is available"
else
  echo "VPC not found"
  exit 1  
fi

echo "Getting subnet details"
if [[ ${PRIVATE_LINK_ENABLED} ]]; then
  echo "Get Private subnet"
  SUBNET=`aws ec2 describe-subnets --region ${CLUSTER_REGION} --filters "Name=vpc-id,Values=$VPC" --query Subnets[0].SubnetId --output text`
else
 echo "Get Public subnet"
 SUBNET=`aws ec2 describe-subnets --region ${CLUSTER_REGION} --filters "Name=vpc-id,Values=$VPC" "Name=tag:Name,Values=*public*" --query Subnets[0].SubnetId --output text`
fi

if [[ $SUBNET =~ "subnet-" ]]; then
 echo "SUBNET "${SUBNET}" is available"
else
 echo "Subnet not found for VPC: ${VPC}"
 exit 1
fi

echo "Getting image: ${IMAGE}"
AMI=`aws ec2 describe-images --filters "Name=name,Values=${IMAGE}" --region ${CLUSTER_REGION} --output text --query 'Images[0].ImageId'`
if [[ $AMI =~ "ami-" ]]; then
  echo "AMI "${AMI}" is available"
else
  echo "AMI not found for ${IMAGE}"
  exit 1 
fi

echo "Creating key-pair"
aws ec2 create-key-pair --key-name ${EC2_NAME}Key --query 'KeyMaterial' --output text --region ${CLUSTER_REGION} > ${EC2_NAME}Key.pem

echo "Creating security group"
SECGRP=`aws ec2 create-security-group --group-name ${EC2_NAME}-sg --description "${EC2_NAME} security group" --vpc-id $VPC --region ${CLUSTER_REGION} --output text`
if [[ $SECGRP =~ "sg-" ]]; then
  echo "SECGRP "${SECGRP}" is created"
fi

echo "Authorising security group"
aws ec2 authorize-security-group-ingress --group-id $SECGRP --protocol tcp --port 22 --cidr 0.0.0.0/0 --region ${CLUSTER_REGION}
sleep 5

if [[ "${FULL_ACCESS_IP}x" != "x" ]]; then
    echo "Setting up ingress rule for ${FULL_ACCESS_IP}"
    aws ec2 authorize-security-group-ingress --group-id ${SECGRP} --protocol all --cidr ${FULL_ACCESS_IP}/32 --region ${CLUSTER_REGION}
    sleep 5
fi

# For Hyperfoil
if [[ "${INSTALL_HYPERFOIL}x" == "truex" ]]; then
    echo "Setting up ingress rule required for Hyperfoil"

    aws ec2 authorize-security-group-ingress --group-id ${SECGRP} --protocol all --cidr 127.0.0.1/32 --region ${CLUSTER_REGION}
    sleep 5
fi

echo "Starting EC2 instance"
EC2=`aws ec2 run-instances --image-id $AMI --count 1 --instance-type ${EC2_TYPE} --key-name ${EC2_NAME}Key --security-group-ids $SECGRP --subnet-id $SUBNET --region ${CLUSTER_REGION} --associate-public-ip-address --block-device-mapping "[ { \"DeviceName\": \"/dev/sda1\", \"Ebs\": { \"VolumeSize\": 50 } } ]" --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${EC2_NAME}}]" --query 'Instances[0].InstanceId' --output text`

aws ec2 wait instance-status-ok --instance-ids ${EC2} --region ${CLUSTER_REGION}

echo "Getting public ip"
PUBIP=`aws ec2 describe-instances --filters "Name=instance-id,Values=$EC2" --query 'Reservations[].Instances[].PublicDnsName' --region ${CLUSTER_REGION} --output text`
if [[ $PUBIP =~ "amazonaws.com" ]]; then
  echo "Public IP "${PUBIP}" is available"
else
  echo "EC2 ${EC2} has been created but it does not seem to have public IP!"
  echo "The key-pair ${EC2_NAME}Key and security group ${SECGRP} have been created so you might want to clean them up!"
  exit 1
fi

INSTANCE=`aws ec2 describe-instances --instance-ids $EC2 --region ${CLUSTER_REGION} --output json`
echo ${INSTANCE}
echo ""
echo "If working locally, you will need to create the ${EC2_NAME}Key.pem and paste in the following contents before running the following commands."
echo ""
cat ${EC2_NAME}Key.pem 
echo ""
echo "chmod 700 ${EC2_NAME}Key.pem"
echo "ssh -i ${EC2_NAME}Key.pem ec2-user@$PUBIP"
