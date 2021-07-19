---
automation:
  - MGDAPI-2336
products:
  - name: rhoam
    environments:
      - external
estimate: 2h
tags:
  - per-release
---

# A40 - RHOAM on OSD Trial

## Description

Verify RHOAM installation on OSD Trial works as expected.

## Steps

1. Provision an OSD Trial Cluster
   - Reach out for AWS credentials needed to provision an OSD Trial Cluster
2. Select RHOAM addon for install on cluster
3. Verify only the `Evaluation` option is available in Quota dropdown
4. Verify `Evaluation` option is the default Quota option
5. Trigger RHOAM installation
6. Maunually add the `useClusterStorage: false` field to the RHMI CR to pass preflight checks and start installation

```
oc patch rhmi rhoam \
        -n redhat-rhoam-operator \
        --type=json \
        -p='[{"op": "add", "path": "/spec/useClusterStorage", "value": "false"}]'
```

7. Verify RHOAM installation completed successfully, and uses the correct `Evaluation` (`0`) quota config

```
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status'
```

7. Trigger uninstall of the addon via the Cluster OCM UI
8. Verify uninstall completes successfully
