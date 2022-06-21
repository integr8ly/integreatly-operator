---
products:
  - name: rhoam
tags:
  - automated
---

# A07 - All operand versions are correct

## Description

Check versions of all products installed by rhmi-operator are correct.

## Steps

1. Get list of expected product versions from `pkg/apis/integreatly/v1alpha1/rhmi_types.go` file in integreatly-operator
2. Get installed versions from cluster
   > oc get rhmi rhmi -n redhat-rhmi-operator
3. Verify installed versions match expected versions
