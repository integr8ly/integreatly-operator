---
estimate: 4h
tags:
  - happy-path
---

# H08 - Verify AMQ Online CRs can be modified and are not reset

## Acceptance Criteria

Verify that fields in the AMQ Online CRs not reconciled by the RHMI Operator can be modified and don't get reset by the RHMI Operator.

AMQ Online CRs:

- AddressPlan
- AddressSpacePlan
- AuthenticationService
- BrokeredInfraConfig
- ConsoleService
- StandardInfraConfig

## Prerequisites

Login to OpenShift console as a **kubeadmin** (user with cluster-admin permissions).

## Steps

1. Go to Projects -> redhat-rhmi-amq-online
2. Home -> Search -> CR-Type -> YAML
3. For each CR type:
   1. Add a new field and value to the CR
   2. Update a value for an existing field in the CR that **is not** reconciled by the rhmi operator. If unsure OSD4 team should be able confirm these fields.
   3. Verify values are not updated in the CR by the rhmi-operator
