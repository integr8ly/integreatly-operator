---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.6.0
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

3. Get the parameter value

```bash
oc get secret addon-managed-api-service-parameters -n redhat-rhoam-operator -o yaml | yq r - 'data.addon-managed-api-service' | base64 --decode
```

Validate that the value of the status.quota matches the parameter from the secret.
