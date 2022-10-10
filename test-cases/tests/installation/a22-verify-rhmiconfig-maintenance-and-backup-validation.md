---
products:
  - name: rhoam
tags:
  - automated
---

# A22 - Verify RHMIConfig maintenance and backup validation

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/rhmi_config.go

## Description

This test is to verify that the RHMIConfig validation webhook for maintenance and backup values works as expected.

## Steps

The expected value formats are:

```
spec:
  backup:
    applyOn: "HH:mm"
  maintenance:
    applyFrom: "DDD HH:mm"
```

1. Add new values in correct format to ensure the validation works with no error. We require the expected values outlined above. We also expect these values to be parsed as a 1 hour window, which should not overlap

```
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/backup/applyOn", "value": "20:00"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/maintenance/applyFrom", "value": "sun 21:00"}]' -n redhat-rhmi-operator
```

2. Add poorly formatted values and ensure that the validation webhook does not allow these changes to be made

```
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/backup/applyOn", "value": "20:000"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/maintenance/applyFrom", "value": "sun 21:000"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/backup/applyOn", "value": "wrong value"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/maintenance/applyFrom", "value": "notADay 12:00"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/backup/applyOn", "value": "wrong value"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/maintenance/applyFrom", "value": "wrong value"}]' -n redhat-rhmi-operator
```

3. Add overlapping values for these time windows and ensure that the validation webhook does not allow these values.

```
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/backup/applyOn", "value": "20:05"}]' -n redhat-rhmi-operator
oc patch rhmiconfig rhmi-config --type=json -p='[{"op" : "add", "path" : "/spec/maintenance/applyFrom", "value": "sun 20:15"}]' -n redhat-rhmi-operator
```
