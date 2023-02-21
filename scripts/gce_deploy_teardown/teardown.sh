#!/bin/bash

set -e

$GCLOUD compute instances delete $VM_NAME --zone $VM_ZONE --quiet
$GCLOUD compute firewall-rules delete $VM_NAME-firewall-rule --quiet

echo "Successfully cleaned up vm instance $VM_NAME and firewall rule $VM_NAME-firewall-rule!"
