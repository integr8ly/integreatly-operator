---
automation:
  - INTLY-6123
components:
  - product-amq
environments:
  - osd-post-upgrade
estimate: 1h
targets:
  - 2.6.0
---

# H08 - Verify AMQ Online CRs can be modified and are not reset

## Acceptance Criteria

Verify that fields in the AMQ Online CRs not reconciled by the RHMI Operator can be modified and don't get reset by the RHMI Operator.

AMQ Online CRs along with fields that can be edited:

- AddressPlan (spec.longDescription)
- AddressSpacePlan (spec.longDescription)
- AuthenticationService (for `none-authservice` change spec.none.certificateSecret.name (make sure that you change it back to the original value after concluding this test))
- BrokeredInfraConfig (spec.admin.resources.memory)
- ConsoleService (spec.discoveryMetadataURL - change from https to http)
- StandardInfraConfig (spec.admin.resources.memory)

## Prerequisites

Login to OpenShift console as a **kubeadmin** (user with cluster-admin permissions).

## Steps

1. Go to Projects -> redhat-rhmi-amq-online
2. Home -> Search -> CR-Type (from the list of CRs above) -> YAML
3. For each CR type:
   1. Add a new field and value to the CR (e.g. new annotation in metadata)
   2. Update a value for an existing field in the CR that **is not** reconciled by the rhmi operator. If unsure OSD4 team should be able confirm these fields. Make sure that you note the original value that you updated and change it back to the original value after finishing the test
   3. Verify values are not updated/ deleted in the CR by the rhmi-operator (wait a couple of minutes for the reconciliation loop to complete and check the values)
