---
products:
  - name: rhmi
  - name: rhoam
estimate: 30m
tags:
  - automated
---

# C03 - Verify that alerting mechanism works

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/alerts_mechanism.go

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Go to the OpenShift cluster Console URL and login as a user with **cluster-admin** role (kubeadmin).
3. Open RHMI Prometheus UI and RHMI Alertmanager (in OpenShift Console, go to `Networking -> Routes` and open `alertmanager-route` URL and `prometheus-route` URL and login using the kubeadmin credentials)

## Steps

1. Trigger a critical severity alert [1] (for `FuseOnlineSyndesisUIInstanceDown` scale down syndesis-operator and syndesis-ui pods). You can do this by opening 2 terminal windows and running the following commands:

```bash
# Terminal window #1
watch "oc scale deployment --replicas=0 syndesis-operator -n redhat-rhmi-fuse-operator"
# Terminal window #2
watch "oc scale deploymentconfig --replicas=0 syndesis-ui -n redhat-rhmi-fuse"
```

**Note**: If you're using a Mac, you can install the `watch` utility by running `brew install watch`

> Verify that the alert starts firing by confirming that the alert in the prometheus UI eventually goes yellow -> red (it can take couple of minutes)
> Verify that the firing alert is also visible in the `Monitoring` tab in the ocm dashboard:

https://qaprodauth.cloud.redhat.com/beta/openshift/details/<cluster_id>

2. Check email, pagerduty and DMS configuration in alertmanager is as expected
   1. Open Alertmanager console
   2. Open `Status` page
      1. Email config should match values in `redhat-rhmi-smtp` secret in `redhat-rhmi-operator` namespace. Use this cmd to get all secret values `for i in $(oc -n redhat-rhmi-operator get secret redhat-rhmi-smtp -o json | jq '.data[]' -r);do echo $i | base64 --decode && printf "\n"; done`
      2. Pagerduty config should match values in `redhat-rhmi-pagerduty` secret in `redhat-rhmi-operator` namespace. Use this cmd to get all secret values `for i in $(oc -n redhat-rhmi-operator get secret redhat-rhmi-pagerduty -o json | jq '.data[]' -r);do echo $i | base64 --decode && printf "\n"; done`
      3. deadmansswitch config should match values in `redhat-rhmi-deadmanssnitch` secret in `redhat-rhmi-operator` namespace. use this cmd to get all secret values `for i in $(oc -n redhat-rhmi-operator get secret redhat-rhmi-deadmanssnitch -o json | jq '.data[]' -r);do echo $i | base64 --decode && printf "\n"; done;`
      4. 2 values are shown as `<secret>` in the Alertmanager UI, these, and all of above, can can be verified in the alertmanager config secret instead using this cmd. `oc -n redhat-rhmi-middleware-monitoring-operator get secret alertmanager-application-monitoring -o json | jq '.data["alertmanager.yaml"]' -r | base64 --decode | grep 'service_key\|smtp\|url'`
3. Terminate the `watch` process in your terminal windows (Ctrl + C) so the pods can be scaled up again (this is done automatically by the RHMI operator).

[1] <https://docs.google.com/spreadsheets/d/1qhZPnWt7kb_ZOoPNCNIbQ7b3Ef9y3UkcICCrIdJM-0U/edit#gid=0>
