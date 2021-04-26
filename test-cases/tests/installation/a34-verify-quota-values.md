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

# A34 - Verify Quota values

## Prerequisites

- Logged in to a testing cluster as a kubeadmin

## Steps

1. Get `toQuota` field's value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.toQuota'
```

> The output should be `null`, unless the quota configuration is still in progress. Try again in couple of minutes.

2. Get the Quota value from RHMI CR

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
```

> The output should be "default" for RHOAM 1.4.0, then FIVE_MILLION_Quota, TWENTY_MILLION_Quota, etc. for next RHOAM
> releases, based on the Quota type selected in OCM
