---
estimate: 15m
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.2.0
      - 1.0.0
      - 1.12.0
      - 1.15.0
---

# A27 - Verify pod priority class is created

## Description

This test case should verify that the pod priority class is for RHOAM with correct value is created

## Steps

1. Log in to OSD cluster via oc

2. Run this command to verify that the value of 'rhoam-pod-priority' priorityClass is set to "1000000000" for single tenant RHOAM install

```bash
oc get priorityclass rhoam-pod-priority -o json | jq -r .value | grep 1000000000 ; echo $?
```

> Verify that you get "0" on the output
