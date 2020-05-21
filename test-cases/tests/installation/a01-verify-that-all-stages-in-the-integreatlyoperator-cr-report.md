---
estimate: 15m
tags:
  - happy-path
---

# A01 - Verify that all stages in the integreatly-operator CR report completed

Acceptance Criteria:

1. The phase of all stages in the Installation CR must report completed
   1. authentication
   2. bootstrap
   3. cloud-resources
   4. monitoring
   5. products
   6. solution-explorer

## Steps

1. Select rhmi-operator namespace
2. Go to Operators section -> Installed Operators
3. Select on RHMI operator
4. Navigate to RHMI Installation tab
5. Select the installation
6. Check status in the YAML
