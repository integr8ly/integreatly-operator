# Additional configuration

If you are running RHOAM against a cluster which is smaller than the requirements listed above, you 
should use the IN_PROW variable, otherwise the installation will not complete. 
If you have a cluster which meets the requirements, this step can be skipped.
Please see the table below for other configuration options.

```shell script
INSTALLATION_TYPE=managed-api IN_PROW=true USE_CLUSTER_STORAGE=<true/false> make deploy/integreatly-rhmi-cr.yml
```

| Variable | Options | Type | Default | Details |
|----------|---------|:----:|---------|-------|
| INSTALLATION_TYPE     | `managed`, `managed-api` or `multitenant-managed-api` | **Required** |`managed`  | Manages installation type. `managed` stands for RHMI. `managed-api` for RHOAM. `multitenant-managed-api` for Multitenant RHOAM. |
| IN_PROW               | `true` or `false`         | Optional      |`false`    | If `true`, reduces the number of pods created. Use for small clusters |
| USE_CLUSTER_STORAGE   | `true` or `false`         | Optional      |`true`     | If `true`, installs application to the cloud provider. Otherwise installs to the OpenShift. |