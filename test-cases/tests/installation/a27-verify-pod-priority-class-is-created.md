---
estimate: 15m
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.2.0
      - 1.0.0
---

# A27 - Verify pod priority class is created

## Description

This test case should verify that the pod priority class is created.

## Steps

1. Log in to cluster console as kubeadmin

2. Search for `Priority Class`

3. Confirm a priority Class called `managed-service-priority` is created

4. Confirm contents are similar to the example below

````kind: PriorityClass
apiVersion: scheduling.k8s.io/v1
metadata:
  name: managed-service-priority
  selfLink: /apis/scheduling.k8s.io/v1/priorityclasses/managed-service-priority
  uid: 84c5b034-0508-415f-a703-71bac56f2d06
  resourceVersion: '155263'
  generation: 1
  creationTimestamp: '2020-10-21T14:44:17Z'
  managedFields:
    - manager: integreatly-operator-local
      operation: Update
      apiVersion: scheduling.k8s.io/v1
      time: '2020-10-21T14:44:17Z'
      fieldsType: FieldsV1
      fieldsV1:
        'f:description': {}
        'f:value': {}
value: 1000000000
description: Priority Class for managed-api```
````
