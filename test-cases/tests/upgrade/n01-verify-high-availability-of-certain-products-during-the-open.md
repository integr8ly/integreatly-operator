---
estimate: 2h
---

# N01 - Verify High Availability of certain products during the OpenShift upgrade

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Node.js installed locally
3. [ocm CLI](https://github.com/openshift-online/ocm-cli/releases) installed locally
4. [jq v1.6](https://github.com/stedolan/jq/releases) installed locally

## Steps

1. Make sure **nobody is using the cluster** for performing the test cases, because the RHMI components will have a downtime during the upgrade
2. In the terminal window #1, git clone the [Delorean repository](https://github.com/integr8ly/delorean) and run the [measure-downtime script](https://github.com/integr8ly/delorean/blob/master/scripts/ocm/measure-downtime.js)

```bash
git clone https://github.com/integr8ly/delorean
cd delorean/scripts/ocm
node measure-downtime.js
```

3. In terminal window #2, run the following command to trigger the OpenShift upgrade

```bash
oc adm upgrade --to-latest=true
```

> You should see the message saying the upgrade of the OpenShift cluster is triggered

4. Login to the ocm staging environment and get the ID of the cluster that is going to be upgraded:

```bash
# Get the token at https://qaprodauth.cloud.redhat.com/openshift/token
ocm login --url=https://api.stage.openshift.com --token=<YOUR-TOKEN>
CLUSTER_ID=$(ocm cluster list | grep <CLUSTER-NAME> | awk '{print $1}')
```

5. Run this command:

```bash
watch -n 60 "ocm get cluster $CLUSTER_ID | jq -r .metrics.upgrade.state | grep -q completed && echo 'Upgrade completed\!'"
```

> This script will run every 60 seconds to check whether the OpenShift upgrade has finished
> Once it's finished, it should print out "Upgrade completed!" (it could take ~1 hour)

6. Go to the OpenShift console, go through the `redhat-rhmi` prefixed namespaces and verify that all routes (Networking -> Routes) of RHMI components are accessible
   > If some of the routes are not accessible, try again later. If they won't come up in the end, report the issue.
7. Terminate the process for measuring the downtime of components in terminal window #1
   > It takes couple of seconds until all results are collected
   > The results will be written down to the file `downtime.json`
8. Upload that file to the JIRA ticket
9. Consult the results with engineering (especially in case some components have a long downtime or are not working properly)
