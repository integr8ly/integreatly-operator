---
environments:
  - osd-post-upgrade
estimate: 120m
tags:
  - destructive
targets:
  - 2.6.0
---

# J03 - Verify that namespaces get recreated by the integreatly-operator if deleted

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/namespace_restoration.go

## Description

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

Note: All but `3scale` namespaces has been automated as pipeline tests due to known bug with 3scale

Test that all namespace will be automatically recreated by the integreatly-operator

Namespaces for manual deletion:

- redhat-rhmi-3scale
- redhat-rhmi-3scale-operator

**Note known bug:** 3scale is being stucked in "in progress" state after ns deletion - workaround: https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/backup_restore/restore_namespace.md#3scale

## Steps

1. By default, this test is not run as part of the functional test suite. To run the test as part of the functional test suite, run the following `makefile` command from the RHMI operator repo against a target cluster:

- **_Note:_** _to speed up this test case, comment out all but destructive tests from the `test/common/tests.go` directory._

  ```
  DESTRUCTIVE=true make test/functional
  ```

2. For every namespace defined in the manual deletion list above:
   1. delete namespace "`oc delete namespace <namespace>`"
   2. check namespace is recreated (e.g. "`oc describe project <namespace>`" / attribute 'Created')
   3. check product is in `Complete` status in RHMI CR
   - If the status is still showing as `in progress` reffer to the **_Bug Notice_** above

**Note finalizers:**

if a namespace stuck in 'Terminating' state, it's needed to remove finalizers to proceed. To find resources with finalizers: "`kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get -n <namespace> --ignore-not-found --show-kind`" Than, to edit resource: "`oc edit <resource> -n <namespace>`"

**Other useful commands here:**

to get stage what is reconciler currently on: `oc get rhmis rhmi -n redhat-rhmi-operator -o json | grep 'stage":'`

to get statuses (+ info) of all stages: `oc get rhmis rhmi -n redhat-rhmi-operator -o json | jq -r '.status.stages'`
