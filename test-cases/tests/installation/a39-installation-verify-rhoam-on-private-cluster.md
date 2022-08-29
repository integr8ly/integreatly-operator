---
products:
  - name: rhoam
    environments:
      - osd-private-post-upgrade
estimate: 3h
tags:
  - per-release
---

# A39 - installation - verify RHOAM on private cluster

## Prerequisites

- access to the [spreadsheet with shared AWS credentials](https://docs.google.com/spreadsheets/d/1P57LhhhvhJOT5y7Y49HlL-7BRcMel7qWWJwAw3JCGMs)

## Steps

### Create a clean OSD cluster as usual

- can be done either manually as described below or via [addon-flow](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow) pipeline
- it is enough for the cluster to be single-AZ

1. Go to the [spreadsheet with shared AWS credentials](https://docs.google.com/spreadsheets/d/1P57LhhhvhJOT5y7Y49HlL-7BRcMel7qWWJwAw3JCGMs) and select "AWS accounts" sheet
2. Look for AWS account ID that is free (doesn't have anything specified in 'Note'). If no account is free, you can use account that is used by nightly pipelines (but don't forget to clean it up for night)
3. Open the [AWS secrets file from 'vault' repository](https://gitlab.cee.redhat.com/integreatly-qe/vault/-/blob/master/SECRETS.md) locally and look for the AWS credentials for the selected AWS account (aws account id, access key ID and secret access key)
4. Go to [OCM UI (staging environment)](https://qaprodauth.cloud.redhat.com/beta/openshift/) and log in
5. Click on `Create cluster` and again `Create cluster` in `Red Hat OpenShift Dedicated` row (not Trial)
6. Click on `Customer cloud subscription` and click `Next`
7. Click on `AWS`, insert AWS account ID, access key ID and secret access key from the SECRETS.md file (from the step above), click on `Validate` and then `Next`
8. Fill in the following parameters and click `Next`:

```
Cluster name: test-ldap-idp
Availability: Multizone
Region: (of your choice, the region must support multi-AZ)
```

9. Fill in 3 (three) for `Worker node count (per zone)` and click `Next`
10. Click `Advanced` for Network configuration, fill in follwing and click `Next`

```
Machine CIDR: 10.11.128.0/24
Service CIDR: 10.11.0.0/18
Pod CIDR: 10.11.64.0/18
Host prefix: /26
```

11. On the "Cluster updates" page just click `Next`
12. Click on `Create cluster` (cluster creation takes ~40 minutes)

Run following command:

```
# Copy your cluster's name from OCM UI ("test-ldap-idp" by default) and assign it to the env var CLUSTER_NAME
CLUSTER_NAME=<your-cluster-name>
# Get cluster's ID
CLUSTER_ID=$(ocm get clusters --parameter search="display_name like '$CLUSTER_NAME'" | jq -r '.items[0].id')
ocm get subs --parameter search="cluster_id = '${CLUSTER_ID}'" | jq -r .items[0].metrics[0].health_state
```

The command should eventually return `healthy`. After cluster is marked as `ready` in OCM UI it should not take more than 30 minutes for this to happen.

13. Follow [this guide](https://docs.google.com/document/d/1BwjzezNFtE7gd2y6FY6v2W6KRXCn0jMZk58ilJ8zSa8/edit) to make it private

### Verify RhOAM installation

1. Login to OCM with the token provided.

```bash
ocm login --url=https://api.stage.openshift.com/ --token=<YOUR_TOKEN>
```

2. Once the cluster is created, you can install the RHOAM addon.

```bash
NOTIFICATION_EMAIL="<your-username>+ID1@redhat.com"
```

```bash
ocm post https://api.stage.openshift.com/api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons --body=<<EOF
{
   "addon":{
      "id":"managed-api-service"
   },
   "parameters":{
      "items":[
         {
            "id":"addon-resource-required",
            "value":"true"
         },
         {
            "id":"cidr-range",
            "value":"10.1.0.0/26"
         },
         {
            "id":"addon-managed-api-service",
            "value":"10"
         },
         {
            "id":"notification-email",
            "value":"$NOTIFICATION_EMAIL"
         }
      ]
   }
}
EOF
```

```
CIDR range: "10.1.0.0/26" (note this down to use it later for another verification step)
Notification email: "<your-username>+ID1@redhat.com <your-username>+ID2@redhat.com"
Quota: 1 Million requests per day
```

3. You should now log in to your cluster via `oc` and patch RHMI CR to select the cloud-storage-type of installation:

```bash
# See above if you do not have CLUSTER_ID populated
# Get your cluster API URL and kubeadmin password
API_URL=$(ocm get cluster $CLUSTER_ID | jq -r .api.url)
KUBEADMIN_PASSWORD=$(ocm get cluster $CLUSTER_ID/credentials | jq -r .admin.password)
# Log in via oc
oc login $API_URL -u kubeadmin -p $KUBEADMIN_PASSWORD --insecure-skip-tls-verify=true
# Patch RHMI CR
oc patch rhmi rhoam -n redhat-rhoam-operator --type=merge -p '{"spec":{"useClusterStorage": "false" }}'
```

4. Now the installation of RHOAM should be in progress. You can watch the status of the installation with this command:

```
watch "oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage"
```

5. Once the status is "completed", the installation is finished and you can go to another step

> Due to a known issue the OCM UI can display `Installed` despite installation still being in progress

### Run the RHOAM functional test suite locally

- navigate to where the [Delorean](https://github.com/integr8ly/delorean) repository is cloned
- `make build/cli`
- create a `test-config.yaml` file with following content

```
---

tests:
- name: integreatly-operator-test
  image: quay.io/integreatly/integreatly-operator-test-harness:rhoam-latest-staging
  timeout: 7200
  envVars:
  - name: DESTRUCTIVE
    value: 'false'
  - name: MULTIAZ
    value: 'false'
  - name: WATCH_NAMESPACE
    value: redhat-rhoam-operator
```

- `KUBECONFIG=<path/to/kubeconfig/file> ./delorean pipeline product-tests --test-config test-config.yaml --output test-results --namespace test-functional | tee testOutput.txt`

- Attach the test results to the ticket, analyze failures if any
