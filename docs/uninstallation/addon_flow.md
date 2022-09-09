# Addon flow
  If you installed RHOAM as an addon then you can uninstall it through the ui as shown in the picture below , or alternatively  you can run the following command. 
```sh
ocm delete /api/clusters_mgmt/v1/clusters/${clusterId}/addons/managed-api-service
```
![Uninstall RHOAM addon](https://user-images.githubusercontent.com/74991829/153239383-52edb7d5-f03a-4b1e-83ca-e5961b2ba577.png)