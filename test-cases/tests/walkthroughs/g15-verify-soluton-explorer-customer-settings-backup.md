---
estimate: 15m
tags:
  - 2.5.0
---

# G15 - Verify soluton explorer customer settings backup

Acceptance Criteria:

When logged into solution explorer as a customer-admin

1. Select ? in the top right hand corner to check the Solution Explorer version. It should be 2.26.1.
2. Select the settings cog in the top right hand corner, it should be active.
3. From the backup drop down menu select a time to do the backup.

When logged into openshift as cluster-admin

1. Goto <openshift-cluster>k8s/ns/redhat-rhmi-operator/integreatly.org~v1alpha1~RHMIConfig/rhmi-config/yaml
2. The time you have selected from the Solution-Explorer should be here.
