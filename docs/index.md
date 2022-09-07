# Integreatly Operator

A Kubernetes Operator based on the Operator SDK for installing and reconciling managed products.

An Integreatly Operator can be installed using three different flavours: `managed`, `managed-api` or `multitenant-managed-api`

To switch between the three you can use export the `INSTALLATION_TYPE` env or use it in conjunction with any of the make commands referenced in this README

## Installed products

The operator installs the following products:

## managed

- AMQ Online
- AMQ Streams
- Codeready
- Fuse
- Nexus
- RHSSO (both a cluster instance, and a user instance)
- 3scale
- Integreatly solution explorer

## managed-api

- 3scale
- RHSSO (both a cluster instance, and a user instance)
- Marin3r

## multitenant-managed-api

- 3scale
- RHSSO (cluster instance)
- Marin3r


