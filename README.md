# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling Integreatly products.

## Current status

This is a PoC / alpha version. Most functionality is there but it is higly likely there are bugs and improvements needed

## Prerequisites
- [Go (v1.12+)](https://golang.org/dl/) 
- [moq](https://github.com/matryer/moq)

### MOQ
After installation, the following commands must be run to avoid a known [issue](https://github.com/matryer/moq/issues/98) related to the package:
```
go get -u .
go install github.com/matryer/moq
```

## Supported Custom Resources

The following custom resources are supported:

- `Installation`

## Local Setup

Create the [`OperatorSource`](https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml) in OpenShift:
```sh
oc create -f https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml
```

Create the Installation `CustomResourceDefinition` in OpenShift:
```sh
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/crds/installation.crd.yaml
```

Create the Namespace/Project for the Integreatly Operator to watch:
```sh
oc new-project <namespace>
```

- Some products will need AWS credentials so create 2 secrets in the Namespace/Project for the Integreatly Operator
    ```
   oc process -f ./deploy/s3-secrets.yaml \
   -p INSTALLATION_NAMESPACE=<test-namespace> \
   -p AWS_ACCESS_KEY_ID=<access key> \
   -p AWS_SECRET_ACCESS_KEY=<access secret> \
   -p AWS_BUCKET=<test bucket> \
   -p AWS_REGION=eu-central-1 | oc apply -f -
    ```
- RHSSO requires a secret called `github-oauth-secret` containing Github OAuth credentials to setup a Github identity provider.

  ```sh
  oc create secret generic github-oauth-secret --from-literal=clientId=<client-id>--from-literal=secret=<client-secret> -n <integreatly-operator-namespace>
  ```

Create the `Installation` resource in the namespace we created:
```sh
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/crds/examples/installation.cr.yaml
```

Create the `Role`, `RoleBinding` and `ServiceAccount`:
```sh
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/service_account.yaml
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/role.yaml
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/role_binding.yaml
```

Create user level sso
```sh
oc new-project mobile-developer-console
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/rhsso-user-role.yaml -n mobile-developer-console
oc create -f https://raw.githubusercontent.com/integr8ly/integreatly-operator/master/deploy/rhsso-user-role_binding.yaml -n mobile-developer-console
```
Set env var
```sh
oc new-project user-sso
oc set env deployment/keycloak-operator 'CONSUMER_NAMESPACES=user-sso; mobile-developer-console' -n user-sso
```

Clone this repository, change directory and run the operator:
```sh
operator-sdk up local --namespace=<namespace>
```

In the OpenShift UI, in `Projects -> integreatly-rhsso -> Networking -> Routes`, select the URL for the `sso` Route to open up the SSO login page.

The username is `admin`, and the password can be retrieved by running:
```sh
oc get dc sso -n integreatly-rhsso -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="SSO_ADMIN_PASSWORD")].value}'
```

## Setting up your cluster to be OSD like

To setup your cluster to have dedicated admins run the ```dedicated-setup.sh``` script
```
cd scripts
./dedicated-setup.sh

```

If you want to remove the opertor run

```bash
cd scripts
./dedicated-cleanup.sh
```

## Deploying to a Cluster with OLM

Create the [`OperatorSource`](https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml) in the cluster:
```sh
oc create -f https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml
```

Create a new namespace to test in, and create the secrets required for 3scale and backups:
```sh
oc process -f ./deploy/s3-secrets.yaml \
   -p INSTALLATION_NAMESPACE=<test-namespace> \
   -p AWS_ACCESS_KEY_ID=<access key> \
   -p AWS_SECRET_ACCESS_KEY=<access secret> \
   -p AWS_BUCKET=<test bucket> \
   -p AWS_REGION=eu-central-1 | oc apply -f -
```

Within a few minutes, the Integreatly operator should be visible in the OperatorHub (`Catalog > OperatorHub`). To create a new subscription, click on the Install button, choose to install the operator in the created namespace and keep the approval strategy on automatic.


Once the subscription shows a status of `installed`, a new Integreatly Installation Custom Resource (CR) can be created which will begin to install the supported services. In `Catalog > Developer Catalog`, choose the Integreatly Installation and click Install. An example installation CR can be found below:

```yml
apiVersion: integreatly.org/v1alpha1
kind: Installation
metadata:
  name: example-installation
spec:
  type: workshop
  namespacePrefix: integreatly-
  routingSubdomain: example.com
  masterUrl: http://master.example.com
  selfSignedCerts: true
```

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

```sh
git tag v<version> && git push upstream v<version>
```


