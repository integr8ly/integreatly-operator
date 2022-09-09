# Local and OLM
If you installed RHOAM locally or through a catalog source then you can uninstall one of two ways:

A) Create a configmap and add a deletion label (Prefered way of uninstallation).
```sh 
oc create configmap managed-api-service -n redhat-rhoam-operator
oc label configmap managed-api-service api.openshift.com/addon-managed-api-service-delete=true -n redhat-rhoam-operator
```

B) Delete the RHOAM cr.
```sh 
oc delete rhmi rhoam -n redhat-rhoam-operator
```

In both scenarios wait until the RHOAM cr is removed and then run the following command to delete the namespace.
```sh 
oc delete namespace redhat-rhoam-operator
```

## Note
After uninstalling RHOAM you should clean up the cluster by running the following command
```sh
export INSTALLATION_TYPE=managed-api
make cluster/cleanup && make cluster/cleanup/crds
```