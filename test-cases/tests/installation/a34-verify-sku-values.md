---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.4.0
estimate: 15m
tags:
  - automated
---

# A34 - Verify SKU values

## Prerequisites

- Logged in to a testing cluster as a kubeadmin

## Steps

1. Get `toSKU` field's value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.toSKU'
```

> The output should be `null`, unless the sku configuration is still in progress. Try again in couple of minutes.

2. Get the SKU value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.sku'
```

> The output should be "default" for RHOAM 1.4.0, then FIVE_MILLION_SKU, TWENTY_MILLION_SKU, etc. for next RHOAM releases, based on the SKU type selected in OCM
