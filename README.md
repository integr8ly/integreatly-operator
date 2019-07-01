# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling Integreatly products.

## Current status

This is a PoC / alpha version. Most functionality is there but it is higly likely there are bugs and improvements needed

## Supported Custom Resources

The following custom resources are supported:

- `Installation`

## Local Setup

- Create the OperatorSource in OpenShift (https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml)
    * `oc create -f https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml`
- Create the Installation CustomResourceDefinition in OpenShift 
    * `oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/crds/installation.crd.yaml`
- Create the Namespace/Project for the Integreatly Operator to watch
    * `oc new-project <namespace>` or `oc create namespace <namespace>`
- Create the Installation resource in the namespace we created
    * `oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/crds/examples/installation.cr.yaml`
- Create the Role, RoleBinding and ServiceAccount
    * `oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/service_account.yaml`
    * `oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/role.yaml`
    * `oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/role_binding.yaml`
- In the integr8ly/integreatly-operator directory, run the operator
    * `operator-sdk up local --namespace=test`
- In the OpenShift Ui, in Projects -> OpenShift-RHSSO -> Networking -> Routes. Select the URL for the `sso` Route to open up the SSO login page.
- The username is `admin`, the password can be retrieved with 
    * `oc get dc sso -n openshift-rhsso -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="SSO_ADMIN_PASSWORD")].value}'`


## Deploying to a Cluster using OLM

- Create the [OperatorSource](https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml) in the cluster
    * `oc create -f https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml`

- The Integreatly Operator should now be available in Operator Hub.

![integreatly operator in hub](./docs/images/integreatly-in-hub.png)

Install the Integreatly Operator

- Then from the Developer catalog choose an Installation custom resource.

![installation custome resource in developer catalog](./docs/images/dev-catalog-installation.png)

and create a desired Installation. There is an example Installation CR at:

https://github.com/integr8ly/integreatly-operator/blob/master/deploy/crds/examples/installation.cr.yaml

## Tests

Running unit tests:

```sh
make test/unit
```

## Release

Update operator version files:

* Bump [operator version](version/version.go)
```Version = "<version>"```
* Bump [makefile TAG](Makefile)
```TAG=<version>```
* Bump [operator image version](deploy/operator.yaml)
```image: quay.io/integreatly/integreatly-operator:v<version>```

Commit changes and open pull request.

When the PR is accepted, create a new release tag:

```git tag v<version> && git push upstream v<version>```


