## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs. 
This docker file builds the root image every time the CI is ran. 
Configuration for the CI-Operator can be found [here](https://docs.ci.openshift.org/docs/architecture/ci-operator/#what-is-ci-operator-and-how-does-it-work).

#### Build and Test

Requires you to be oc logged in to any cluster on your host machine.

`git` is a requirement of the ci-operator.

```
$ docker build -t registry.svc.ci.openshift.org/integr8ly/intly-operator-base-image:latest - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/integr8ly/intly-operator-base-image:latest test/run
operator-sdk version: "v1.11.0", commit: "215fc50b2d4acc7d92b36828f42d7d1ae212015c", kubernetes version: "v1.18.8", go version: "go1.13.5", GOOS: "linux", GOARCH: "amd64"
go version go1.13.5 linux/amd64
jq-1.6
yq version 4.9.8
git version 1.8.3.1
v12.16.3
Delorean CLI
...
SUCCESS!
```

### ../.ci-operator.yaml

Contains the imagestream that is used to build the root image.
Reference: [https://docs.ci.openshift.org/docs/architecture/ci-operator/#build-root-image](https://docs.ci.openshift.org/docs/architecture/ci-operator/#build-root-image)

### Updating

To update the version of the root images edit the `FROM` image in the Dockerfile.tools and update the imagestream in the .ci-operator.yaml.
The ci-operator will work from the HEAD of the PR and build the new updated images.
There is no need to do any edits in the `openshift/release` repo to get these changes added.

