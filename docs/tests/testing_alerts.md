# Testing Alerts

The ability to trigger Critical and Warning alerts exists.
This is done by the creation of a secret in the observability operator namespace.

## Test Fire Severity Alerts

### Critical Alert
Create a generic secret named `cj3cssrec` to fire Critical Alert (TestFireCriticalAlert).
```shell
oc create secret generic cj3cssrec -n local-rhoam-observability
```
It should take < 3 minutes for the alert to appear.
The alert that will be firing is called **TestFireCriticalAlert**

Deleting the secret will stop the alert from firing.
It will take a few minutes for the alert to resolve itself.  
```shell
oc delete secret cj3cssrec -n local-rhoam-observability
```

### Warning Alert
Create a generic secret named `wj3cssrew` to fire Critical Alert (TestFireWarningAlert).
```shell
oc create secret generic wj3cssrew -n local-rhoam-observability
```
It should take < 3 minutes for the alert to appear.
The alert that will be firing is called **TestFireWarningAlert**

Deleting the secret will stop the alert from firing.
It will take a few minutes for the alert to resolve itself.
```shell
oc delete secret wj3cssrew -n local-rhoam-observability
```