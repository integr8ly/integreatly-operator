#!/bin/bash

# Usage: disable and renable an AZ
#
# prereq
#  - jq
#  - aws-cli

# AZ to disable
AZ=eu-west-1a

# Checks the command line params true/nothing to disable and false to re-enable
DISABLE=$1
if [ -z "$DISABLE" ]; then
 	echo "No option set expect params true/false, the default option is disable which is true"
 	echo "true - disable the AvailabilityZone"
 	echo "false - restore the original configuration (run only after a successful disable)"
 	DISABLE=true
fi

# Function changeAcl takes two params for disable or enable
# $1 should be NetworkAclAssociationId filename
# $2 should be NetworkAclId filename
function ChangeAcl() {
	count=1
	cat $1 | while read NetworkAclAssociationId
	do
		echo $(sed -n "${count}p" < NetworkAclId.tmp)
		echo $NetworkAclAssociationId
		aws ec2 replace-network-acl-association --association-id $NetworkAclAssociationId --network-acl-id $(sed -n "${count}p" < $2)
	    ((count=count+1))
	done
}

if $DISABLE
then
  echo "Disabling AvailabilityZone"
  # use the subnetId to get the NetworkAclAssociationId to create the new acl association and get the NetworkAclId so can revert the change
  for SUBNETID in $(aws ec2 describe-subnets | jq ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.SubnetId')
  do
    aws ec2 describe-network-acls | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclAssociationId' >> NetworkAclAssociationId.tmp
      aws ec2 describe-network-acls | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclId' >> NetworkAclId-restore.tmp
  done

  # create two the dummy ACL and create a file containing the NetworkAclId for the dummy ACL
  for VPCID in $(aws ec2 describe-subnets | jq -r ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.VpcId')
  do
    aws ec2 create-network-acl --vpc-id $VPCID | jq -r '.NetworkAcl.NetworkAclId' >> NetworkAclId.tmp
  done

  # create new disable ACL association
  ChangeAcl NetworkAclAssociationId.tmp NetworkAclId.tmp
else
  echo "Re-enable AvailabilityZone"
  for SUBNETID in $(aws ec2 describe-subnets | jq ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.SubnetId')
  do
    aws ec2 describe-network-acls | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclAssociationId' >> NetworkAclAssociationId-restore.tmp
  done

  # Restore the subnets to the original ACL's
  ChangeAcl NetworkAclAssociationId-restore.tmp NetworkAclId-restore.tmp


  # delete the dummy ACL's
  cat NetworkAclId.tmp | while read deleteNetworkAclId
  do
    aws ec2 delete-network-acl --network-acl-id $deleteNetworkAclId
  done

  # remove the tmp files
  rm *.tmp
fi