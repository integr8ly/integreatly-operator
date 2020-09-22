## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs.

#### Build and Test

```
$ docker build -t registry.svc.ci.openshift.org/integr8ly/intly-operator-base-image:latest - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/integr8ly/intly-operator-base-image:latest test/run
operator-sdk version: "v0.17.1", commit: "6d108056f39fa19546cf9235fad6e84b0683114a", kubernetes version: "v1.17.2", go version: "go1.13.5 linux/amd64"
go version go1.13.5 linux/amd64
jq-1.6
yq version 3.1.2
v12.16.3
Delorean CLI
...
SUCCESS!
```