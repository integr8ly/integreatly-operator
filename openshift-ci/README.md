## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs.

#### Build and Test

```
$ docker build -t registry.svc.ci.openshift.org/openshift/release:intly-golang-1.12 - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/openshift/release:intly-golang-1.12 test/run 
operator-sdk version: v0.8.1, commit: 33b3bfe10176f8647f5354516fff29dea42b6342
go version go1.12.9 linux/amd64
SUCCESS!
```