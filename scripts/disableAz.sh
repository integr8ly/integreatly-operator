#!/bin/bash

# Usage: disable and renable an AZ
# ./disableAz.sh <true/false> <aws availability zone>
# prereq
#  - jq
#  - aws-cli

# check the correct command line args are passed two expected
if (( $# < 2 )); then
  echo "Incorrect amount of arguments passed in"
 	echo "usage ./disableAz.sh <true/false> <aws az>"
 	exit 1
fi

# Checks the command line args $1 true to disable and false to re-enable
DISABLE=$1
if [[ "$DISABLE" == "true" ]] || [[ "$DISABLE" == "false" ]]; then
  echo "Arg $1 accepted"
else
 	echo "Incorrect option for first arg expect true/false"
 	echo "usage ./disableAz.sh <true/false> <aws az>"
 	echo "- true - disable the AvailabilityZone"
 	echo "- false - restore the original configuration (run only after a successful disable)"
 	exit 1
fi

# Sets the command line args $2 for the availability zone
AZ=$2
# check $2 for matching az string
AZCHECK=`echo $AZ | grep "eu-west-\|eu-central-\|eu-north-\|eu-south-\|us-east-\|us-west-\|ap-northeast-\|ap-south-\|ap-southeast-|\sa-east-\|ca-central-"`
# if the $2 arg does not match the stings above then
if [ -z "$AZCHECK" ]; then
 	echo "Incorrect availability zone in args"
 	echo "AvailabilityZone examples"
 	echo "- eu-west-1a"
 	echo "- eu-west-1b"
 	echo "- eu-west-1c"
 	echo "- eu-central-1a"
 	echo "etc ...."
 	echo " "
 	echo "usage ./disableAz.sh <true/false> <aws az>"
 	exit 1
fi


# Function changeAcl takes two arguments for disable or enable
# $1 should be NetworkAclAssociationId filename
# $2 should be NetworkAclId filename
function ChangeAcl() {
	count=1
	cat $1 | while read NetworkAclAssociationId
	do
		echo $(sed -n "${count}p" < NetworkAclId.tmp)
		echo $NetworkAclAssociationId
		aws ec2 replace-network-acl-association --region ${AZ%?} --association-id $NetworkAclAssociationId --network-acl-id $(sed -n "${count}p" < $2)
	    ((count=count+1))
	done
}

if $DISABLE; then
  echo "Disabling AvailabilityZone"
  # remove existing files if exists
  if [ -f "NetworkAclAssociationId.tmp" ];  then
    echo "Removing existing NetworkAclAssociationId.tmp file"
    rm NetworkAclAssociationId.tmp
  fi
  if [ -f "NetworkAclId-restore.tmp" ];  then
    echo "Removing existing NetworkAclId-restore.tmp file"
    rm NetworkAclId-restore.tmp
  fi
  if [ -f "NetworkAclId.tmp" ];  then
    echo "Removing existing NetworkAclId.tmp file"
    rm NetworkAclId.tmp
  fi

  # use the subnetId to get the NetworkAclAssociationId to create the new acl association and get the NetworkAclId so can revert the change
  for SUBNETID in $(aws ec2 describe-subnets --region ${AZ%?}| jq ".Subnets[] | select(.AvailabilityZone==\"$AZ\")"  | jq -r '.SubnetId')
  do
    aws ec2 describe-network-acls --region ${AZ%?}| jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclAssociationId' >> NetworkAclAssociationId.tmp
    aws ec2 describe-network-acls --region ${AZ%?}| jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclId' >> NetworkAclId-restore.tmp
  done

  # create two the dummy ACL and create a file containing the NetworkAclId for the dummy ACL
  for VPCID in $(aws ec2 describe-subnets --region ${AZ%?} | jq -r ".Subnets[] | select(.AvailabilityZone==\"$AZ\")"  | jq -r '.VpcId')
  do
    aws ec2 create-network-acl --vpc-id $VPCID --region ${AZ%?} | jq -r '.NetworkAcl.NetworkAclId' >> NetworkAclId.tmp
  done

  # create new disable ACL association
  ChangeAcl NetworkAclAssociationId.tmp NetworkAclId.tmp
else
  echo "Re-enable AvailabilityZone"
  for SUBNETID in $(aws ec2 describe-subnets --region ${AZ%?} | jq ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.SubnetId')
  do
    aws ec2 describe-network-acls --region ${AZ:?} | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclAssociationId' >> NetworkAclAssociationId-restore.tmp
  done

  # Restore the subnets to the original ACL's
  ChangeAcl NetworkAclAssociationId-restore.tmp NetworkAclId-restore.tmp


  # delete the dummy ACL's
  cat NetworkAclId.tmp | while read deleteNetworkAclId
  do
    aws ec2 delete-network-acl --network-acl-id $deleteNetworkAclId --region ${AZ%?}
  done

  # remove the tmp files
  rm NetworkAclAssociationId-restore.tmp NetworkAclId-restore.tmp NetworkAclAssociationId.tmp NetworkAclId.tmp
fi