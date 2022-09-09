# Getting started guide

Please ensure the pre-requisites are met and your cluster matches the required size

## Prerequisites

- [operator-sdk](https://github.com/operator-framework/operator-sdk) version v1.21.0.
- [go](https://golang.org/dl/) version 1.18+
- [moq](https://github.com/matryer/moq)
- [oc](https://docs.okd.io/latest/cli_reference/openshift_cli/getting-started-cli.html) version v4.6+
- [yq](https://github.com/mikefarah/yq) version v4+
- [jq](https://github.com/stedolan/jq)   
- Access to an Openshift v4.6.0+ cluster
- A user with administrative privileges in the OpenShift cluster

After installation, the following commands must be run to avoid a known issue related to the Moq package:
```shell
make code/compile
go install github.com/matryer/moq
```

## Cluster size guidelines

For development work the required vcpu and ram can be lower than that stated in the [service definition](https://access.redhat.com/articles/5534341#scalability-and-service-levels-15).
Different quotas require different values.
Table belong are typical requested values needed for RHOAM on a cluster with cluster storage set to True.

| Quota        | vCPU     | RAM     |
|--------------|----------|---------|
| 100 Thousand | 6.5 vCPU | 22 Gb   |
| 1 Million    | 6.5 vCPU | 22 Gb   |
| 5 Million    | 8 vCPU   | 24 Gb   |
| 10 Million   | 8.5 vCPU | 24 Gb   |
| 20 Million   | 9.5 vCPU | 24 Gb   |
| 50 Million   | 14 vCPU  | 26.5 Gb |