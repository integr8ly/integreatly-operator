---
products:
  - name: rhoam
    environments:
      - external
estimate: 4h
tags:
  - manual-selection
---

# A38 - installation - verify RHOAM on non-byoc cluster

Obsolete test case
More information: https://issues.redhat.com/browse/MGDAPI-2397

## Prerequisites

- access to ENG OCM ORG (ocm account with "\_rhmi" suffix)

## Steps

1. Go to [OCM UI (staging environment)](https://qaprodauth.cloud.redhat.com/beta/openshift/) and log in
2. Click on `Create cluster` and select `Red Hat OpenShift Dedicated`
3. Select AWS and click on `Standard`
4. Fill in the following parameters:

```
Cluster name: test-nonbyoc
Availability: Single zone
Worker node count: 6
Networking: Basic
```

5. Click on `Create cluster` (cluster creation takes ~40 minutes)

**Verify RHOAM installation via addon**

1. Once the cluster is created, you can install the RHOAM addon
2. Select your cluster -> `Add-ons` and click on `Install`
3. Fill in the following parameters and click on `Install`

```
CIDR range: "10.1.0.0/26" (note this down to use it later for another verification step)
Notification email: "cloud-services-qe-reporting@redhat.com"
```

4. You should now login to your cluster via `oc` and patch RHMI CR to select the cloud-storage-type of installation:

```bash
# Copy your cluster's name from OCM UI ("test-ldap-idp" by default) and assign it to the env var CLUSTER_NAME
CLUSTER_NAME=<your-cluster-name>
# Get cluster's CID
CID=$(ocm get clusters --parameter search="display_name like '$CLUSTER_NAME'" | jq -r '.items[0].id')
# Get your cluster API URL and kubeadmin password
API_URL=$(ocm get cluster $CID | jq -r .api.url)
KUBEADMIN_PASSWORD=$(ocm get cluster $CID/credentials | jq -r .admin.password)
# Log in via oc
oc login $API_URL -u kubeadmin -p $KUBEADMIN_PASSWORD --insecure-skip-tls-verify=true
# Patch RHMI CR
oc patch rhmi rhoam -n redhat-rhoam-operator --type=merge -p '{"spec":{"useClusterStorage": "false" }}'
```

5. Now the installation of RHOAM should be in progress. You can watch the status of the installation with this command:

```
watch "oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage"
```

6. Once the status is "completed", the installation is finished and you can go to another step

> Due to a known issue the OCM UI can display `Installed` despite installation still being in progress

**Run automated tests**

1. Navigate to [RHOAM addon flow jenkins job](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow)
2. Fill in the following parameters

```
rhmiReleaseVersion: 1.x.x-rcx (replace x.x-rcx with minor and patch version of the current release and rc number)
ocmAccessToken: paste your ocm token (for eng org)
clusterId: test-nonbyoc
stepsToDo: select "IDP" and "tests" checkbox (untick everything else)
```

3. Click on "Build"
4. Once the job is finished, navigate to the build number of the job -> Build Artifacts -> results -> integreatly-operator-tests -> logs -> container.log
5. Scroll down and check if there are some skipped/failed tests. If so, rerun them manually. If they are still failing, report the issue in JIRA (see the JIRA epic description for more info)
