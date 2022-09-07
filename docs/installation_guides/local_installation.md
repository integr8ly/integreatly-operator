# Local Development
Ensure that the cluster satisfies minimal requirements: 
- RHMI (managed): 26 vCPU 
- RHOAM (managed-api and multitenant-managed-api): 18 vCPU. More details can be found in the [service definition](https://access.redhat.com/articles/5534341) 
  under the "Resource Requirements" section

## Clone the integreatly-operator
Only if you haven't already cloned. Otherwise, navigate to an existing copy. 
```sh
mkdir -p $GOPATH/src/github.com/integr8ly
cd $GOPATH/src/github.com/integr8ly
git clone https://github.com/integr8ly/integreatly-operator
cd integreatly-operator
```

## Prepare your cluster

If you are working against a fresh cluster it will need to be prepared using the following. 
Ensure you are logged into a cluster by `oc whoami`.
Include the `INSTALLATION_TYPE`. See [here](#3-configuration-optional) about this and other optional configuration variables.
```shell
INSTALLATION_TYPE=<managed/managed-api> make cluster/prepare/local
```

## Run integreatly-operator
Include the `INSTALLATION_TYPE` if you haven't already exported it. 
The operator can now be run locally:
```shell
INSTALLATION_TYPE=<managed/managed-api/multitenant-managed-api> make code/run
```
If you want to run the operator from a specific image, you can specify the image and run `make cluster/deploy`
```shell
IMAGE_FORMAT=<image-registry-address> INSTALLATION_TYPE=managed-api  make cluster/deploy
```

*Note:* if the operator doesn't find an RHMI cr, it will create one (Name: `rhmi/rhoam`).

| Variable | Options | Type | Default | Details |
|----------|---------|:----:|---------|-------|
| PRODUCT_DECLARATION | File path | Optional |`./products/installation.yaml` | Specifies how RHOAM install the product operators, either from a local manifest, an index, or an included bundle. Only applicable to RHOAM |