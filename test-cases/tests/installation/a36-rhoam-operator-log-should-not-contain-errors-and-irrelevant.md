---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.4.0
estimate: 15m
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
