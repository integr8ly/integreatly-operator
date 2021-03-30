---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.4.0
estimate: 30m
---

# H27 - Verify that user with uppercase letters can be created in 3scale

## Description

This test verifies that if there is an existing user with uppercase letters in the name in OpenShift, that user can be also created in 3scale (during RHOAM installation). The username in 3scale should be with lowercase letters

## Prerequisites

- Clean OSD cluster
- Github IDP
- Github user with at least one uppercase letter in the username

## Steps

**Set up Github IDP for OSD cluster**

1. Go to https://qaprodauth.cloud.redhat.com/beta/openshift/ -> select your cluster -> Access control -> Add identity providers
2. Fill in the details (Client ID, Client Secret, add "integr8ly" organization)
3. Log in to your cluster via Github IDP
4. Verify that the user you've logged in with has an uppercase letters in its name

```bash
oc get users | awk '{print $1}' | grep -i <your-username>
```

5. Trigger RHOAM installation via OCM UI
6. You should now login to your cluster via `oc` and patch RHMI CR to select the cloud-storage-type of installation:

```bash
# Copy your cluster's name from OCM UI and assign it to the env var CLUSTER_NAME
CLUSTER_NAME=<your-cluster-name>
# Get cluster's CID
CID=$(ocm get clusters --parameter search="display_name like '$CLUSTER_NAME'" | jq -r '.items[0].id')
# Get your cluster API URL and kubeadmin password
API_URL=$(ocm get cluster $CID | jq -r .api.url)
KUBEADMIN_PASSWORD=$(ocm get cluster $CID/credentials | jq -r .admin.password)
# Log in via oc
oc login $API_URL -u kubeadmin -p $KUBEADMIN_PASSWORD --insecure-skip-tls-verify=true
# Patch RHMI CR
oc patch rhmi rhoam -n redhat-rhoam-operator --type=merge -p '{"spec":{"useClusterStorage": "true" }}'
```

5. Now the installation of RHOAM should be in progress. You can watch the status of the installation with this command:

```
watch "oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage"
```

> Verify that the installation is finished (it should take ~40 minutes and you should see "completed" in the output)

6. In OpenShift console (when logged in as your github user), select the launcher on the top right menu -> API Management -> Github IDP and log in to 3scale
   > Verify that you can successfully log in
7. Go to Account settings (top right menu) -> Personal -> Personal Details
   > Verify that your username contains only lowercase letters
