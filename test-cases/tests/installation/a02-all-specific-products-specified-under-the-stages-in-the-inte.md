---
tags:
  - happy-path
estimate: 15m
---

# A02 - All specific products specified under the stages in the integreatly-operator CR have reported completed

## Prerequisites

Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

Note: to get above command, login web console as kubeadmin, click its name in top-right corner and then 'Copy Login Command'

## Steps

1. Run the command below. It gets all RHMI components and gets their phase and filter them based on the phase (status).

```
oc get rhmis rhmi -n redhat-rhmi-operator -o json | jq -r '.status.stages' | grep '"status": "completed"' | wc -l
```

> You should see '13' in the output, which is the number of RHMI products which have 'completed' status:
>
> 1.  RHSSO
> 2.  Cloud Resources
> 3.  Monitoring
> 4.  Apicurito
> 5.  3scale
> 6.  AMQ Online
> 7.  Codeready Workspaces
> 8.  Data Sync
> 9.  Fuse
> 10. Fuse on OpenShift
> 11. User RHSSO
> 12. UPS
> 13. Solution Explorer

2. If the above is not true, run the command below and check for the component which doesn't have 'completed' phase and investigate/create an issue.

```
oc get rhmis rhmi -n redhat-rhmi-operator -o json | jq -r '.status.stages'
```
