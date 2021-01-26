#!/bin/bash

echo "Creating key-pair"
aws ec2 create-key-pair --key-name ${EC2_NAME}Key --query 'KeyMaterial' --output text --region ${CLUSTER_REGION} > ${EC2_NAME}Key.pem && Keyfile=true

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
SECGRP=`aws ec2 create-security-group --group-name ${EC2_NAME}-sg --description "${EC2_NAME} security group" --vpc-id $VPC --region ${CLUSTER_REGION} --output text`
for (( i=0; i<5; i++ )) do
    if [[ $SECGRP =~ "sg-" ]]; then
      echo "SECGRP "${SECGRP}" is created"
      break
    fi
    sleep 5
done

# For Hyperfoil
if [[ "${INSTALL_HYPERFOIL}x" == "truex" ]]; then
    echo "Setting up ingress rules required for Hyperfoil"
    if [[ "${FULL_ACCESS_IP}x" != "x" ]]; then
        aws ec2 authorize-security-group-ingress --group-id ${SECGRP} --protocol all --cidr ${FULL_ACCESS_IP}/32 --region ${CLUSTER_REGION}
    fi

    aws ec2 authorize-security-group-ingress --group-id ${SECGRP} --protocol all --cidr 127.0.0.1/32 --region ${CLUSTER_REGION}
fi

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

echo "Starting EC2 instance"
EC2=`aws ec2 run-instances --image-id $AMI --count 1 --instance-type ${EC2_TYPE} --key-name ${EC2_NAME}Key --security-group-ids $SECGRP --subnet-id $SUBNET --region ${CLUSTER_REGION} --associate-public-ip-address --block-device-mapping "[ { \"DeviceName\": \"/dev/sda1\", \"Ebs\": { \"VolumeSize\": 50 } } ]" --query 'Instances[0].InstanceId' --output text`

aws ec2 wait instance-status-ok --instance-ids ${EC2} --region ${CLUSTER_REGION}

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
echo "If working locally, you will need to create the ${EC2_NAME}Key.pem and paste in the following contents before running the following commands."
echo ""
cat ${EC2_NAME}Key.pem 
echo ""
echo "chmod 700 ${EC2_NAME}Key.pem"
echo "ssh -i ${EC2_NAME}Key.pem ec2-user@$PUBIP"
