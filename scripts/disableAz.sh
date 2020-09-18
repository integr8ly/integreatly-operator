#!/bin/bash

# prereq jq
# DISABLE=$1
# if [ -z "$DISABLE" ]; then
# 	echo No option set expect prama true/false, the default option is disable which is true
# 	$DISABLE=true
# fi


AZ=eu-west-1c

# function changeAcl takes two params for disable or enable
# $1 should be NetworkAclAssociationId filename
# $2 should be NetworkAclId filename
function changeAcl() {
	count=1
	cat $1 | while read NetworkAclAssociationId
	do
		echo $count
		echo $(sed -n "${count}p" < NetworkAclId.tmp)
		echo $NetworkAclAssociationId
		aws ec2 replace-network-acl-association --association-id $NetworkAclAssociationId --network-acl-id $(sed -n "${count}p" < $2)
	    ((count=count+1))
	done
}

# use the subnetId to get the NetworkAclAssociationId to create the new acl accociation and get the NetworkAclId so can revert the change
for SUBNETID in $(aws ec2 describe-subnets | jq ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.SubnetId')
do
	aws ec2 describe-network-acls | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclAssociationId' >> NetworkAclAssociationId.tmp
    aws ec2 describe-network-acls | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclId' >> NetworkAclId-restore.tmp
done


# if [[ $DISABLE ]]

# create two the dummy ACL and create a file containing the NetworkAclId for the dummy ACL
for VPCID in $(aws ec2 describe-subnets | jq -r ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.VpcId')
do
	aws ec2 create-network-acl --vpc-id $VPCID | jq -r '.NetworkAcl.NetworkAclId' >> NetworkAclId.tmp
done



# create new disable ACL accociation
changeAcl NetworkAclAssociationId.tmp NetworkAclId.tmp


sleep 1m
for SUBNETID in $(aws ec2 describe-subnets | jq ".Subnets[] | select(.AvailabilityZone==\"$AZ\")" | jq -r '.SubnetId')
do
	aws ec2 describe-network-acls | jq -r ".[] | .[].Associations[] | select(.SubnetId==\"$SUBNETID\")" | jq -r '.NetworkAclAssociationId' >> NetworkAclAssociationId-restore.tmp
done

# Restore the subnets to the orignal ACL's
changeAcl NetworkAclAssociationId-restore.tmp NetworkAclId-restore.tmp

# delete the dummy ACL's
cat NetworkAclId.tmp | while read deleteNetworkAclId
do
	aws ec2 delete-network-acl --network-acl-id $deleteNetworkAclId
done

# remove the tmp files
rm *.tmp