# Deploy and Remove EC2 instances on AWS

The scripts will be used to deploy and remove EC2 instances where hyperfoil.io will be deployed for managed-api performance tests.

For further information contact Tony Davidson

## The following pipelines are available.

1. ec2-deploy-remove https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/ManagedAPI/job/ec2-deploy-remove/
2. ec2-deploy https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/ManagedAPI/job/ec2-deploy/
3. ec2-remove https://master-jenkins-csb-intly.cloud.paas.psi.redhat.com/job/ManagedAPI/job/ec2-remove/

## Usage and Paramaters.

### Ec2-deploy-remove Usage

The *ec2-deploy-remove* pipeline will deploy and immediately remove an ec2 instance and is intended for future development.

### Ec2-deploy Usage and Parameters

The *ec2-deploy* pipeline will create an EC2 instance on an aws account where an OSD cluster is deployed.

The following parameters are required

1. **EC2integreatlyOperatorOrg** - defaults to *integreatly* - The repository where the deployEC2.sh script resides.
2. **EC2integreatlyOperatorBranchName** - defaults to *master* - The branch of the repository you wish to use.
3. **EC2customAwsAccessKeyId** - no default - *REQUIRED* - the osdCcsAdmin AWS_ACCESS_KEY_ID for the AWS account where the OSD cluster is deployed.
4. **EC2customAwsSecretAccessKey** - no default - *REQUIRED* - the osdCcsAdmin AWS_SECRET_ACCESS_KEY for the AWS account where the OSD cluster is deployed.
5. **EC2clusterID** - no default - *REQUIRED* - the cluster id and build number that was used to create the OSD cluster. For example : \<your cluster id\>-\<build number\>, tdavidso-31.
6. **EC2awsRegion** - no default - *REQUIRED* - the region in which the OSD cluster was provisioned. This can be obtained using the following commands in a terminal.

    ocm login --url=https://api.stage.openshift.com/ --token=\<your ocm token\>
    
    ocm get /api/clusters_mgmt/v1/clusters/$(ocm get clusters | jq '.items[] | select(.name | startswith("\<your cluster id\>")).id' -r)  | jq -r .region.id
7. **EC2image** - defaults to *RHEL-8.2.0_HVM-*-x86_64*Hourly2-GP2* - The AMI image on AWS used to create the EC2 instance.

### Ec2-remove Usage and Parameters

The *ec2-remove* pipeline will remove an EC2 instance on an aws account where an OSD cluster is deployed.

The following parameters are required

1. **EC2integreatlyOperatorOrg** - defaults to *integreatly* - The repository where the tearDown.sh script resides.
2. **EC2integreatlyOperatorBranchName** - defaults to *master* - The branch of the repository you wish to use.
3. **EC2customAwsAccessKeyId** - no default - *REQUIRED* - the osdCcsAdmin AWS_ACCESS_KEY_ID for the AWS account where the OSD cluster is deployed.
4. **EC2customAwsSecretAccessKey** - no default - *REQUIRED* - the osdCcsAdmin AWS_SECRET_ACCESS_KEY for the AWS account where the OSD cluster is deployed.
5. **EC2clusterID** - no default - *REQUIRED* - the cluster id and build number that was used to create the OSD cluster. For example : \<your cluster id\>-\<build number\>, tdavidso-31.
6. **EC2awsRegion** - no default - *REQUIRED* - the region in which the OSD cluster was provisioned. This can be obtained using the following commands in a terminal.

    ocm login --url=https://api.stage.openshift.com/ --token=\<your ocm token\>
    
    ocm get /api/clusters_mgmt/v1/clusters/$(ocm get clusters | jq '.items[] | select(.name | startswith("\<your cluster id\>")).id' -r)  | jq -r .region.id
