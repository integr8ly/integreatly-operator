---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.5.0
      - 1.8.0
      - 1.11.0
      - 1.14.0
      - 1.19.0
      - 1.22.0
      - 1.25.0
      - 1.28.0
      - 1.32.0
      - 1.35.0
      - 1.38.0
      - 1.41.0
estimate: 120m
tags:
  - destructive
---

# J03B - Verify that namespaces get recreated by the integreatly-operator if deleted

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/namespace_restoration.go

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

Test that all namespace will be automatically recreated by the integreatly-operator

Note: known issue that prevent this test case to be fully executed automatically: https://issues.redhat.com/browse/MGDAPI-4456

## Steps

1. Login via `oc` as **kubeadmin**

2. By default, this test is not run as part of the functional test suite. To run this singular functional test, run the following command from the RHOAM operator repo against a target cluster:

```
LOCAL=false DESTRUCTIVE=true INSTALLATION_TYPE=managed-api TEST="J03" make test/e2e/single | tee test-results.log
```

3. Check the namespaces in RHOAM except the `redhat-rhoam-operator` are recreated during the test run, the Active for
   namespaces from the command below should be recent.

```
oc get ns | grep rhoam
redhat-rhoam-3scale                                Active   1h
redhat-rhoam-3scale-operator                       Active   1h
redhat-rhoam-cloud-resources-operator              Active   1h
redhat-rhoam-customer-monitoring                   Active   1h
redhat-rhoam-marin3r                               Active   1h
redhat-rhoam-marin3r-operator                      Active   1h
redhat-rhoam-operator                              Active   5d13h
redhat-rhoam-operator-observability                Active   1h
redhat-rhoam-rhsso                                 Active   1h
redhat-rhoam-rhsso-operator                        Active   1h
redhat-rhoam-user-sso                              Active   1h
redhat-rhoam-user-sso-operator                     Active   1h
```

**Note finalizers:**

If a namespace stuck in 'Terminating' state, it's needed to remove finalizers to proceed. To find resources with finalizers:

```
kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get -n <namespace> --ignore-not-found --show-kind
```

Then, to edit resource: `oc edit <resource> -n <namespace>`

**Other useful commands here:**

to get stage what is reconciler currently on: `oc get rhmis rhoam -n redhat-rhoam-operator -o json | grep 'stage":'`

to get statuses (+ info) of all stages: `oc get rhmis rhoam -n redhat-rhoam-operator -o json | jq -r '.status.stages'`
