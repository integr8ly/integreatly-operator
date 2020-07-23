---
---

# C12 - Verify no critical alerts have fired on cluster during lifespan of cluster

## Prerequisites

- access to `integreatly-notifications@redhat.com` email list
- workload webapp should be running on cluster and have been deployed shortly after cluster was provisioned

## Description

Verify no critical alerts have fired on cluster during its lifespan.

This should be one of the last testcases performed on a cluster to allow for maximum burn-in time on cluster.

Testcase should not be performed on a cluster that has been used for destructive testing.

## Steps

1. Check `integreatly-notifications@redhat.com` for any alerts that have fired on this cluster
2. If any critical alerts have fired:
   > Take screenshots showing the time the alerts fired and when they were resolved  
   > Create a followup bug JIRA and inform release coordinators. Example JIRA: https://issues.redhat.com/browse/INTLY-9443  
   > Request that cluster lifespan be extended to allow time for cluster to be investigated (ask release coordinator).
