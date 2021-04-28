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

Note: There is also an automated version of this test, so the steps below are just initial checks to validate the quota
parameter is as expected and to validate an upgrade scenario. Please also check that the A34 automated test is passing.

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

3. Get the param value from the Secret

```bash
oc get secret addon-managed-api-service-parameters -n redhat-rhoam-operator -o json | jq -r '.data.addon-managed-api-service' | base64 --decode
```

Verify that the quota value matches the parameter from the secret.

If there is no value in the secret, the cluster has been upgraded from a version of rhoam which did not have
the quota paramater. aka pre 1.6.0. If this is the case go to step 4.

4. Get the param value from the container Environment Variable.

```bash
oc get $(oc get pods -n redhat-rhoam-operator -o name | grep rhmi-operator) -n redhat-rhoam-operator -o json | jq -r '.spec.containers[0].env[] | select(.name=="QUOTA")'
```
