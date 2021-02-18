## OpenShift CI

### Dockerfile.tools

Base image used on CI for all builds and test jobs.

#### Build and Test

```
$ docker build -t registry.svc.ci.openshift.org/integr8ly/intly-operator-base-image:latest - < Dockerfile.tools
$ IMAGE_NAME=registry.svc.ci.openshift.org/integr8ly/intly-operator-base-image:latest test/run
operator-sdk version: "v1.2.0", commit: "215fc50b2d4acc7d92b36828f42d7d1ae212015c", kubernetes version: "v1.18.8", go version: "go1.13.5", GOOS: "linux", GOARCH: "amd64"
go version go1.13.5 linux/amd64
jq-1.6
yq version 3.1.2
v12.16.3
Delorean CLI
...
SUCCESS!
```