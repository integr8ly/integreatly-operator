# 3scale guides
This section covers 3scale guides

## Creating 3scale backends through CR`s

This guide documents the creation of backends through Custom Resources by CLI and Openshift UI

 > **NOTE**: If a backend is created through a CR then update and delete operations should be done through the CR only

### Prerequisites

- Created 3scale tenant (in order to get providerAccountRef)
- At least customer admin permissions to create a backend in your namespace
- Backends have to be created in the same namespace as the tenant

### Create a 3scale backend
- CLI 
  1. Create backend CR 
    
    ```sh
    kubectl apply -f - <<EOF
    ---
    apiVersion: capabilities.3scale.net/v1beta1
    kind: Backend
    metadata:
      name: <CR_Name>
      namespace: <3scale_Tenant_Namespace>
    spec:
      name: "<3scale_Backend_Name>"
      systemName: "<3scale_Backend_System_Name>"
      privateBaseURL: "<PrivateBaseURL>"
      providerAccountRef:
        name: <Tenant_Secret_Name>
    EOF
    ```

- UI 
  1. As kubeadmin in the openshift console navigate to Home > Search
  2. click on the Resources drop-down menu, search for backend
  3. Click on the blue Create Backend button
  4. Replace the contents with the following
  ```
  apiVersion: capabilities.3scale.net/v1beta1
    kind: Backend
    metadata:
      name: <CR_Name>
      namespace: <3scale_Tenant_Namespace>
    spec:
      name: "<3scale_Backend_Name>"
      systemName: "<3scale_Backend_System_Name>"
      privateBaseURL: "<PrivateBaseURL>"
      providerAccountRef:
        name: <Tenant_Secret_Name>
  ```

### Sample Values

|     Variable     |   Example   | Method  | Explanation |
| ------------- | ------------- | -------------  | -------------- |
| <CR_Name> | backend1-sample | Set  | Name of backend CR |
|  <3scale_Backend_Name> | backend1  | Set | Name of backend in 3scale |
|  <3scale_Backend_System_Name> | backend  | Set | System name for backend in 3scale |
| <PrivateBaseURL.> | https://api.example.com  | Set | Url for backend |
| <Tenant_Secret_Name> | tenant1ref | Get  | Name assigned to `<Tenant_Secret_name>` during tenant creation , this secret contains the tenant url and access token so we know which tenant the backend should be created in.
| <3scale_Tenant_Namespace> | 3scale-test | Get | Has to be the same namespace that was used for tenant creation 


### Sample CR

```
apiVersion: capabilities.3scale.net/v1beta1
kind: Backend
metadata: 
  name: backend1-sample
  namespace: 3scale-test
spec: 
  name: "backend1"
  systemName: "backend"
  privateBaseURL: "https://api.example.com"
  providerAccountRef: 
    name: tenant1ref
```

## Creating 3scale products through CR`s

This guide documents the creation of products through Custom Resources by CLI and Openshift UI

> **NOTE**: If a product is created through a CR then update and delete operations should be done through the CR only

### Prerequisites

- Created 3scale tenant (in order to get providerAccountRef)
- At least customer admin permissions to create a product in your namespace
- Products have to be created in the same namespace as the tenant

### Create a 3scale product

- CLI
  1. Create product CR
    
    ```sh
    kubectl apply -f - <<EOF
    ---
    apiVersion: capabilities.3scale.net/v1beta1
    kind: Product
    metadata:
      name: <CR_Name>
      namespace: <3scale_Tenant_Namespace>
    spec:
      name: "<3scale_Product_Name>"
      providerAccountRef:
        name: <Tenant_Secret_Name>
    EOF
    ```

- UI 
  1. As kubeadmin in the openshift console navigate to Home > Search
  2. click on the Resources drop-down menu, search for product
  3. Click on the blue Create Product button
  4. Replace the contents with the following
  ```
  apiVersion: capabilities.3scale.net/v1beta1
    kind: Product
    metadata:
      name: <CR_Name>
      namespace: <3scale_Tenant_Namespace>
    spec:
      name: "<3scale_Product_Name>"
      providerAccountRef:
        name: <Tenant_Secret_Name>
  ```


### Sample Values

|     Variable     |   Example   | Method  | Explanation |
| ------------- | ------------- | -------------  | -------------- |
| <CR_Name> | product1-sample | Set  | Name of product CR |
|  <3scale_Product_Name> | product1  | Set | Name of product in 3scale |
| <Tenant_Secret_Name> | tenant1ref | Get  | Name assigned to `<Tenant_Secret_name>` during tenant creation, this secret contains the tenant url and access token so we know which tenant the product should be created in.
| <3scale_Tenant_Namespace> | 3scale-test | Get | Has to be the same namespace that was used for tenant creation


### Sample CR

```
apiVersion: capabilities.3scale.net/v1beta1
kind: Product
metadata: 
  name: product1-sample
  namespace: 3scale-test
spec: 
  name: "product1"
  providerAccountRef:
    name: tenant1ref
```

### Specifying backend in product

To specify a backend usage in a product add the following under product.spec in the product CR
```
backendUsages:
    <backend_name>:
      path: /
```

## Creating 3scale tenants through CR`s

This guide documents the creation of tenants through Custom Resources by CLI and Openshift UI

> **NOTE**: If a tenant is created through a CR then update and delete operations should be done through the CR only

### Prerequisite

- At least customer admin permissions to create namespace and tenant in that namespace

### Creating namespace for tenant

Create a new namespace for 3scale-tenant

  - CLI
    ```sh
    oc new-project <3scale_Tenant_Namespace>
    ```

  - UI
    1. As kubeadmin in the openshift console navigate to Home > Projects
    2. In projects click on the blue Create Project button, fill in the form and create the project



### Create passwordCredentialsRef for tenant

Create passwordCredentialsRef to use as a reference in tenant

  - CLI
    ```sh
    oc create secret generic <PassCredRef_Secret_Name> --from-literal=admin_password=<Password> -n <3scale_Tenant_Namespace>
    ```

  - UI 
     1. As kubeadmin in the openshift console navigate to Workloads > Secrets
     2. Make sure that the project in the top left is the same as the project namespace you created above
     3. In secrets click  the blue Create button and select From YAML
     4. Replace the contents with the following
    ```
    apiVersion: v1
    kind: Secret
    metadata:
      name: <PassCredRef_Secret_Name>
      namespace: <3scale_Tenant_Namespace>
    type: Opaque
    stringData:
      admin_password: <Password>
    ```



### Create a 3scale tenant

- CLI
  1. Retrieve master URL
    ```
    oc get routes -n redhat-rhoam-3scale | grep master
    ```

  2. Create tenant CR 
    
    ```sh
    kubectl apply -f - <<EOF
    ---
    apiVersion: capabilities.3scale.net/v1alpha1
    kind: Tenant
    metadata:
      name: <CR_Name>
      namespace: <3scale_Tenant_Namespace>
    spec:
      email: <Tenant_email>
      masterCredentialsRef:
        name: system-seed
        namespace: redhat-rhoam-3scale
      organizationName: <Tenant_Org>
      passwordCredentialsRef:
        name: <PassCredRef_Secret_Name>
        namespace: <3scale_Tenant_Namespace>
      systemMasterUrl: https://<Master_Url>
      tenantSecretRef:
        name: <Tenant_Secret_Name>
        namespace: <3scale_Tenant_Namespace>
      username: <Tenant_Username>
    EOF
    ```


- UI
  1. As kubeadmin in the openshift console navigate to Networking > Routes
  2. Make sure that the project in the top left is the same as the project namespace you created in the beggining
  3. Find the system master service and copy the location URL
  4. Navigate to Home > Search, click on the Resources drop-down menu, search for tenant
  5. Click on the blue Create Tenant button
  6. Replace the contents with the following
  ```
  apiVersion: capabilities.3scale.net/v1alpha1
    kind: Tenant
    metadata:
      name: <CR_Name>
      namespace: <3scale_Tenant_Namespace>
    spec:
      email: <Tenant_email>
      masterCredentialsRef:
        name: system-seed
        namespace: redhat-rhoam-3scale
      organizationName: <Tenant_Org>
      passwordCredentialsRef:
        name: <PassCredRef_Secret_Name>
        namespace: <3scale_Tenant_Namespace>
      systemMasterUrl: https://<Master_Url>
      tenantSecretRef:
        name: <Tenant_Secret_Name>
        namespace: <3scale_Tenant_Namespace>
      username: <Tenant_Username>
    ```



### Sample Values

|     Variable     |   Example   | Method  | Explanation |
| ------------- | ------------- | -------------  | -------------- |
| <3scale_Tenant_Namespace> | "3scale-test" | Set | Name of namespace which will be used to add tenant,products and backends and necessary secrets |
|<PassCredRef_Secret_Name> | "passcredref" | Set | Name of secret which will be used as credential reference when creating tenant , secret should contain one key value `admin_password:<password>` |
| <Password.> | password | Set |Password which will be used to login to tenant account |
| <CR_Name> | tenant1-sample | Set | Name of tenant CR
| <Tenant_email> | tenant@mail.com | Set | Email for tenant
| <Tenant_Org> | tenant1-rrg | Set | Organization Name for tenant in 3scale
| <Master_Url> | master.apps.pstefans.ooq8.s1.devshift.org | Get | [Master Url](./tenant_creation_via_cr.md/#create-a-3scale-tenant) for 3scale. This url will be used to create a url for the 3scale tenant.
|<Tenant_Secret_Name> | tenant1ref | Set | Name of secret which will be automatically created with the tenant url and access token.This secret will be used as a providerAccountRef when creating products and backends. 
|<Tenant_Username> | tenant1 | Set | Name of tenant in 3scale


### Sample CR

```
apiVersion: capabilities.3scale.net/v1alpha1
kind: Tenant
metadata:
  name: tenant1-sample
  namespace: 3scale-test
spec:
  email: tenant1@mail.com
  masterCredentialsRef:
    name: system-seed
    namespace: 3scale-test
  organizationName: tenant1-org
  passwordCredentialsRef:
    name: passcredref
    namespace: 3scale-test
  systemMasterUrl: 'https://master.apps.pstefans.ooq8.s1.devshift.org'
  tenantSecretRef:
   name: tenant1ref
   namespace: 3scale-test
  username: tenant1
```

## Known Issues

This document will list some of the know issues with Backend, Product and Tenant creation and steps to fix the issue.

### Product

|     Error     |   Reason   | Fix |
| ------------- | ------------- | -------- |
|In product CR status.conditons : `Task failed SyncBackendUsage: Backend SystemName backend1 not found in  3scale backend index`         | Backend was removed through UI and Product CR is still using it in `backendUsages` | Delete the backend CR , remove the backendusages from product spec  in the product CR and save the product CR. This will fix the error and delete the backend properly |

## Validate installation 

Use following commands to validate that installation succeeded:

For `RHMI` (managed): `oc get rhmi rhmi -n redhat-rhmi-operator -o json | jq .status.stage`

For `RHOAM` (managed-api): `oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage `

For `RHOAM Multitenant` (multitenant-managed-api): `oc get rhmi rhoam -n sandbox-rhoam-operator -o json | jq .status.stage `

Once the installation completed the command wil result in following output:  
```yaml
"complete"
```