# Adding new product
The goal of this guide is to show how to add a new product's operator and CR to the range of operator's managed by the
integreatly-operator. This will touch on a few different areas of the code-base, and explain the purpose of each area.

### Terminology
This guide assumes familiarity with Openshift and Kubernetes terminology, and terminology specific to the Integreatly-operator
is covered [here](../terminology.md).

### Areas of code-base to modify
- Add manifests files for the new operator to `manifests/` directory.
- The product variables to the `api/v1alpha1/rhmi_types.go` file.
- Add product to applicable installation types.
- A new reconciler for the product in the `pkg/products` directory.
- Update reconciler factory.
- A new config for the product in the `pkg/config` directory.
- Update config manager.

### Add Manifest Files
Every product has an operator, and every operator is installed and maintained via OLM. To enable a particular version of
the integreatly-operator to always install a specific version of each product, it maintains it's own set of manifests 
for each product. To do this:
- Create a new directory in the `manifests` directory, named: `integreatly-<product-name>`
- Create a `<product-name>.package.yaml` file, the channel defined in this file must be `rhmi`.
- Create a directory for the release of the operator to be deployed `<product>-<version>`.
- Copy all the manifest files for the operator into this directory.

**Note:** The operator installs the operator and product in separate namespaces and leverages the operatorgroup
functionality of OLM to do this. In order to be compatible with how OLM does this the new CSV must define the namespace
watched by the operator like so:
```asciidoc
- name: <WATCH_NAMESPACE_ENV_VAR>
  valueFrom:
    fieldRef:
      fieldPath: metadata.annotations['olm.targetNamespaces']
``` 

### Define Product Variables 
Every product has a series of variables defined in `api/v1alpha1/rhmi_types.go`, which are used through
out the code-base, you will also need to define these for the new product.
- ProductName this must be DNS-valid (i.e. all lower-case, and only alphanumeric and dashes)
- ProductVersion
- OperatorVersion

### Add Product to Applicable Installation Types
For a product to be installed as part of an installation type, the configuration of that installation type needs to be
updated to include the new product. These installation types are defined in `pkg/controller/installation/types.go`, at
the time of writing, there are 2 installation types defined in variables in this file:
- managed defined in `allManagedStages`
- workshop defined in `allWorkshopStages`

### Deciding when to create a new stage
New stages are not desirable, as the operator will not progress to the following phase, until everything in the current
phase has reported that it has completed, so adding new stages can slow down installation time.

The idea of a stage is to allow the operator to complete some action, that will be required in the following stage, if 
product A is required by product B, product A should probably go in an earlier stage than product B. For example, RHSSO
requires cloud resource, so the cloud resources operator had to go into it's own stage, prior to the stage that RHSSO is 
in.

In general, if the new product is another tool for developers to use, it probably belongs in the products stage. Where 
you add it inside a stage is immaterial; as these are not processed in any particular order, and nothing in a stage 
should depend on something else in the same stage.

### New Product Reconciler
The reconciler must implement the `Products.Interface` interface, in order to work with  the installation controller. 
The methods are defined in more detail below:

### Reconcile
This is the primary entry point of your reconciler, on each reconcile loop of the integreatly-operator,
all the logic in here should be written with the assumption that resources may or may not exist, and can
be created or updated based on their current state. 

This is called every time the stage containing this reconciler is processed, the stages are processed any time a 
watched CRD is modified, or alternatively, every 10 minutes - and all preceding stages are complete.

### Useful Notes on Reconciler design
A few hints and tips that may come in useful when putting a new reconciler together.
 
### Helper Library: Resources
There are quite a lot of helpful functions in the resource package for handling the more common tasks, such as setting 
up the CSV, catalog source, subscription and namespaces. Many examples of this are available in the other reconcilers 
(such as [RHSSO](https://github.com/integr8ly/integreatly-operator/blob/master/pkg/products/rhsso/reconciler.go), etc). 

There is also a helper function to add a finalizer to the RHMI CR, this also requires a function allowing the organisation 
of the tear-down of the product at uninstall time, again examples of this can are available in the existing reconcilers.

### Authentication
Our approach to authentication for products consoles has been to authenticate against Openshift where possible. If the
product is unable to do this, we have set up RHSSO (not to be confused with rhssouser) to federate identities which can 
also be used for authentication by products.

We base our authentication model on 2 groups of users:

rhmi-developers: Every user on the cluster is added to the rhmi-developers group in the
[user_controller](https://github.com/integr8ly/integreatly-operator/blob/master/internal/controller/user/user_controller.go). Which allows us to apply general cluster-wide permissions 
for all users to this group.

dedicated-admins: This group is maintained by OSD, and is modified through the OSD web console, or via the OCM CLI tool.
any user in this group is reconciled into any integreatly product as an admin in that product (the only exception to 
this is the RHSSO which federates openshift users for product authentication; no users can log in to this product).

### Custom Types for Operator's CRDs
When interacting with a products CRDs, it is required to be able to manipulate them in code. Where the products operator
is written in go, these can be imported using [go mod](https://golang.org/ref/mod), rather than copying them into our 
own code-base, making it far easier to keep them updated in the future.

### Parameters
There are several parameters passed into this function:

#### ctx
 This must be used in all network requests performed by the reconciler, as the integreatly-operator maintains this context
 and may kill it if an uninstall is detected.
 
#### installation
This is the RHMI CR we are basing the install from, it has values that are occasionally required by reconcilers, for example
the namespace prefix. 

#### product
This is a pointer to the this reconciler's product in the status block of the CR, it can be used to set values
such as version, host and operator version.

#### serverClient
This is the k8s client to the cluster, and is used for getting, creating, updating and deleting resources in the cluster.

### Return Values
The return values from this method are `state` and `err`:
### State
This is communicated back to the user via the status block of the RHMI CR, the potential values are defined 
[here](https://github.com/integr8ly/integreatly-operator/blob/master/api/v1alpha1/rhmi_types.go) and this field is most commonly either in progress or complete. 
It can go to `fail` if something has broken, but this will not prevent the installation_controller
from calling the Reconcile function in the future, which may allow the reconciler to fix whatever issue had
occurred (i.e. the service had not come up yet, so there were network errors accessing it's API).

### Err
This is how we can communicate to the user via the status block of the RHMI CR what is causing a product to 
enter a failed state, and is written into the `status.lastError` of the CR along with any other errors from
other reconcilers.

### Events
The resources package also contains an events package to help reconcilers handle emitting events. In the case of an 
error, the package has an `events.HandleError` method. In the case of a completed installation,  there is a 
`events.HandleProductComplete`.

### GetPreflightObject
Before the integreatly-operator will begin an installation, it will initially check the cluster has no existing installs
of the same products; this is to avoid potential issues of 2 operators trying to act on one resource. 

This function informs the operator of what object it should look for, to check if the product is already installed. The 
namespace argument is the namespace currently being scanned for existing installations.

For example, codeready looks for a deployment in the scanned namespace with the name "codeready", if found this 
installation will stall until that product is removed.

### Update Reconciler Factory
The [reconciler factory](https://github.com/integr8ly/integreatly-operator/blob/master/pkg/products/reconciler.go) is used by the installation_controller to build your reconciler 
when it comes across your product in the installation type. It follows a fairly simple pattern for most products.

### Create a Config Object for the Product
Each product has a config object, this is used for 2 purposes:
1. The config key/value pairs are persisted in a configmap, so should the operator crash and restart, the values are 
still available.
2. The config of one product can be read from the reconciler of another product (e.g. getting realm and namespace of the 
cluster SSO).

The config object exists in the `pkg/config` directory and must satisfy the `Config.ConfigReadable` interface. The 
methods of this interface are expanded on below:

### Read() ProductConfig
This is used by the configManager to convert your config to yaml and store it in the configmap.

### GetProductName() integreatlyv1alpha1.ProductName
This must return the value of the variable defined earlier in ProductName

### GetProductVersion() integreatlyv1alpha1.ProductVersion
This must return the value of the variable defined earlier in ProductVersion

### GetOperatorVersion() integreatlyv1alpha1.`OperatorVersion
This must return the value of the variable defined earlier in OperatorVersion

### GetHost() string
Return a URL that can be used to access the product, either an API, or console, or blank if not applicable.

### GetWatchableCRDs() []runtime.Object
This should return an array of CRDs that should be watched by the integreatly-operator, if a change of one of these CRDs
in the product's operand namespace is detected, it will trigger a full reconcile of the integreatly-operator. This 
usually returns all of the CRDs the new products operator watches.

### GetNamespace() string
This should return the namespace that the product will be installed into.

### Update Config Manager
The [config manager](https://github.com/integr8ly/integreatly-operator/blob/master/pkg/config/manager.go) is used by the installation_controller, and by the reconcilers, to read the config of products. This
needs to be updated to know how interact with the new products config object. To do this the configManager requires the 
`ReadProduct` function updated with the new config, and also a new function named as: `Read<ProductName>`.

### Add Types to Scheme
Open the [api/v1alpha1/addtoscheme_integreatly_v1alpha1.go](https://github.com/integr8ly/integreatly-operator/blob/master/api/v1alpha1/addtoscheme_integreatly_v1alpha1.go) file and add the product operator types to the Scheme so the components can map objects to GroupVersionKinds and back.

### Tests
We include both unit tests and e2e tests in the integreatly-operator, the unit tests are run on every PR. The e2e tests
are also run on every PR, in all the nightly builds, and as validation of the operator against the OSD managed-tenants
repository, so it is essential that these are all kept current and valid.

### Unit Tests
The unit tests are written using go's standard testing framework, and using the idiomatic format of defining test-cases
in a slice and then iterating over the test-cases using `t.Run`. This allows us to test for race conditions (though we 
do not test for this yet) and run the tests concurrently. Please look at the unit tests for some of the other reconcilers
for examples of how to implement them in more technical detail, such as the [RHSSO tests](https://github.com/integr8ly/integreatly-operator/blob/master/pkg/products/rhsso/reconciler_test.go)
and the [monitoring tests](https://github.com/integr8ly/integreatly-operator/blob/master/pkg/products/monitoring/reconciler_test.go).

### E2E Tests
These are written to be compatible with 2 testing frameworks, due to this, there are 2 entries points for the e2e testing
frameworks which ultimately lead to the same set of tests being executed. The E2E tests are all contained in the 
[tests directory](https://github.com/integr8ly/integreatly-operator/tree/master/test). This directory contains a [BEST_PRACTICES.md](https://github.com/integr8ly/integreatly-operator/blob/master/test/BEST_PRACTICES.md) which should be 
referred to when designing a test.

When creating a test, it is easiest to create it in the common directory, though it is not required. Once the test is
created it needs to be added to the suite of tests executed in an e2e test [here](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/tests.go), note that
the prefixed codes are not required, they are used to track which tests which are currently manual have been convered to
e2e tests.

