---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.4.0
      - 1.9.0
estimate: 15m
tags:
  - manual-selection
  - destructive
---

# A35 - 3scale DeploymentConfig should rescale if it failed during installation

## Prerequisites

- Logged in to a testing cluster as a kubeadmin

## Steps

1. Scale down RHOAM operator

```bash
oc scale deployment rhmi-operator --replicas 0 -n redhat-rhoam-operator
```

2. Manually cause system-app DC to fail

```bash
oc rollout latest dc/system-app -n redhat-rhoam-3scale
sleep 10
PRE_HOOK_POD=$(oc get pod -n redhat-rhoam-3scale -o name | grep "hook-pre" | tail -n1)
oc delete $PRE_HOOK_POD -n redhat-rhoam-3scale
```

3. Verify system-app-deploy pod is in an error state

```bash
LAST_DEPLOY_POD=$(oc get pods -n redhat-rhoam-3scale -o name | grep "system-app.*-deploy" | tail -n1)
oc get $LAST_DEPLOY_POD -n redhat-rhoam-3scale
```

4. Verify system-app deployment config has a 'False' condition due to ReplicationController failed progressing

```bash
oc get dc system-app -n redhat-rhoam-3scale -o json | jq '.status.conditions'
```

3. Bring RHOAM operator back up again

```bash
oc scale deployment rhmi-operator --replicas 1 -n redhat-rhoam-operator
```

4. Wait for a while (~2 minutes, until 3scale is reconciled) and verify that new system-app deployment is rolled out due to failed condition in RHOAM operator log

```bash
oc logs $(oc get pods -n redhat-rhoam-operator -o name | grep rhmi-operator) -n redhat-rhoam-operator | grep "3scale dc in a failed condition"
```

> You should see `WARN[2021-03-10T15:56:35Z] 3scale dc in a failed condition, rolling out new deployment dc=system-app product=3scale` in the output

5. Verify rollout completes successfully

```bash
oc get pods -n redhat-rhoam-3scale | grep "system-app.*deploy" | tail -n1
```

> The status should be 'Completed'

6. Verify RHOAM CR is in complete state

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq '.status.stage'
```
