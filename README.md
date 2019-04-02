# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling Integreatly products.

## Current status

This is a PoC / alpha version. Most functionality is there but it is higly likely there are bugs and improvements needed

## Supported Custom Resources

The following custom resources are supported:

- `Installation`

## Test it locally

TODO

## Deploying to a Cluster

TODO

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


