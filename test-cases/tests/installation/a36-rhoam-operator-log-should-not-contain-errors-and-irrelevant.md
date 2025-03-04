---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.4.0
      - 1.7.0
      - 1.10.0
      - 1.13.0
      - 1.16.0
      - 1.20.0
      - 1.23.0
      - 1.26.0
      - 1.29.0
      - 1.32.0
      - 1.35.0
      - 1.38.0
estimate: 15m
tags:
  - manual-selection
---

# A36 - RHOAM operator log should not contain errors and irrelevant warnings

## Prerequisites

- Logged in to a testing cluster as a kubeadmin

## Steps

**`Required number of routes do not exist` warning should not be present** in the log

1. Search for `Required number of routes do not exist` in the rhmi operator log

```bash
oc logs $(oc get pods -n redhat-rhoam-operator -o name | grep rhmi-operator) -n redhat-rhoam-operator | grep "Required number of routes do not exist"
echo $?
```

> The output should be "1"
