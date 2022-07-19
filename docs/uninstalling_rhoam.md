
## Uninstalling RHOAM
This section covers uninstallation of RHOAM if it was installed via locally, OLM or on ROSA

### Local and OLM installation type
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

#### Note: After uninstalling RHOAM you should clean up the cluster by running the following command.
```sh
export INSTALLATION_TYPE=managed-api
make cluster/cleanup && make cluster/cleanup/crds
```

### Addon
  If you installed RHOAM as an addon then you can uninstall it through the ui as shown in the picture below , or alternatively  you can run the following command. 
```sh
ocm delete /api/clusters_mgmt/v1/clusters/${clusterId}/addons/managed-api-service
```
![Uninstall RHOAM addon](https://user-images.githubusercontent.com/74991829/153239383-52edb7d5-f03a-4b1e-83ca-e5961b2ba577.png)


### ROSA Addon
  If you installed RHOAM as an addon on [ROSA](https://cloud.redhat.com/products/amazon-openshift) then you can uninstall it by running the following command.
```sh 
rosa uninstall addon \
--cluster=${clusterName} managed-api-service -y
```