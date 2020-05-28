---
estimate: 15m
---

# A23 - Verify rhmi operator limits

This test is to verify that the rhmi-operator had resources limits applied

The expected output of rhmi-operator pod yaml config (`spec.template.spec.resources`) should be:

```
limits:
  cpu: 80m
  memory: 1536Mi
requests:
  cpu: 40m
  memory: 64Mi
```
