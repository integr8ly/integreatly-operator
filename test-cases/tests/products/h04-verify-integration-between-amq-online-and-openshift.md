---
automation:
  - INTLY-7432
components:
  - product-amq
  - openshift
environments:
  - osd-post-upgrade
estimate: 30m
targets:
  - 2.8.0
---

# H04 - Verify integration between AMQ Online and OpenShift

## Prerequisites

Login as user in the **developer** group.

## Steps

1. Open the Solution Explorer and start AMQ Online
   > The shared namespace for AMQ should be created in OpenShift (ex.: username-shared-45ce)
   >
   > Create a new addressspace under the new shared namespace
   >
   > The addressspace should be created in OpenShift
   >
   > ```
   > oc get addressspaces  --namespace=<the-shared-namespace>
   > ```
2. Open the AMQ Online Shared Console and create a new Address
   > The address should be successfully created in AMQ Online
   >
   > The address should be created in OpenShift
   >
   > ```
   > oc get addresses  --namespace=<the-shared-namespace>
   > ```
3. Delete the Address
   > The address should be deleted from AMQ and OpenShift
4. Go to the AMQ Online Console and delete the Addressspace
   > The addressspace should be removed from AMQ and OpenShift
