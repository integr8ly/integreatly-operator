---
products:
  - name: rhoam
automation:
  - INTLY-5441
tags:
  - automated
---

# H03 - Verify integration between 3Scale and OpenShift

## Prerequisites

Login as user in the **dedicated-admin** group.

## Steps

1. Open the 3Scale Console and create a new dummy API ( New Product )
   > The API should be created in 3scale
   >
   > Two new routers should be created in OpenShift
   >
   > - {api-name}-3scale-apicast-**staging**.apps.{cluster-domain}
   > - {api-name}-3scale-apicast-**production**.apps.{cluster-domain}
   >
   > ```
   > oc get routes --namespace=redhat-rhmi-3scale | grep '{API_NAME}-3scale-apicast-'
   > ```
2. Delete the dummy API (Overview -> Edit -> Delete)
   > The API should be deleted from 3scale and the routers form OpenShift
