# ROSA
  If you installed RHOAM as an addon on [ROSA](https://cloud.redhat.com/products/amazon-openshift) then you can uninstall it by running the following command.
```sh 
rosa uninstall addon \
--cluster=${clusterName} managed-api-service -y
```