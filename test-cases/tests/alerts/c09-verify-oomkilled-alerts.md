---
targets:
  - 2.3.0
---

# C09 - Verify OOMKilled alerts

## Description

Verify that there are 2 tiers of alerts for OOMKilled pods:

- Warning for when at least 1 OOMKilled is detected
- Critical if the rate of being killed is greater than X

Also verify that those alerts only target RHMI namespaces and that there is an SOP for the critical alert.

More info: <https://issues.redhat.com/browse/INTLY-7778>

## Steps

### Check that alerts are triggered for RHMI OOMKilled pods

1. Open OpenShift console in your browser
2. Login as admin
3. Open terminal for one of RHMI containers that has memory limits set (e.g. `address-space-controller` in `redhat-rhmi-amq-online` namespace)
4. `export BYTES=$((1024*1024*1024))`
5. Fill memory with `yes | tr \\n x | head -c $BYTES | grep n`
   > Pod should be OOMKilled
6. Check Alerts in RHMI Prometheus console (find route in `redhat-rhmi-middleware-monitoring-operator` namespace)
   > Warning alert for detecting OOMKilled pods should be firing
7. Make the pod be OOMKilled again X times (instead of manually filling memory every time, you can also use `polinux/stress` image mentioned below)
   > Critical alert for detecting OOMKilled pods should be firing

Another alternative to make pod be OOMKilled in RHMI namespace is to use `polinux/stress` image, so instead of using above command to fill memory in one of the RHMI containers, you can simply use: `oc run generate-ooms -n redhat-rhmi-3scale --image='polinux/stress' --limits='memory=100Mi' --command -- "stress" "--vm" "1" "--vm-bytes" "101M" "--vm-hang" "1200"` (a small side effect might be that the pod crashlooping alert will also fire). Then when we want the alert to stop firing: `oc delete pod generate-ooms -n redhat-rhmi-3scale`.

### Check that alerts are NOT fired for non-RHMI OOMKilled pods

1. Create new project in OpenShift
2. Under `Workloads` tab click `add another content`
3. Select `From Catalog`
4. Search for `node`
5. Select `Node.js + MongoDB`
6. `Instantiate` and `Create`
7. Navigate to `Workloads > Pods`
8. Wait for `nodejs-mongo-persistent` pod to be ready
9. Open the pod's terminal
10. Use the above 2 commands to fill the container's memory
    > Pod should be OOMKilled
11. Check Alerts in RHMI Prometheus console
    > Warning alert for detecting OOMKilled pods should NOT be firing

### Check that there is an SOP link for the critical alert

1. Open the critical alert for detecting OOMKilled pods in RHMI Prometheus
   > Verify that there is a link to SOP
2. Follow the SOP to check that it is correct
