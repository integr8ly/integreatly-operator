---
targets:
  - 2.6.0
---

# M05 - Verify secrets

More info:

- <https://issues.redhat.com/browse/INTLY-4885>
- <https://issues.redhat.com/browse/INTLY-9096>

## Steps

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
3. Open **Workloads > Secrets**
4. Click on `alertmanager-application-monitoring` secret
5. **Reveal Values**
6. Make note of value for `receivers[1].pagerduty_configs.service_key`
7. Navigate to `redhat-rhmi-operator` namespace
8. Open **Workloads > Secrets**
9. Click on `redhat-rhmi-pagerduty` secret
10. **Reveal Values**
    > Value for `serviceKey` key should match value noted earlier
11. **Actions > Edit Secret**
12. **Add Key/Value**
13. For **Key** use `PAGERDUTY_KEY`
14. For **Value** use `new_test`
15. Check again `alertmanager-application-monitoring` secret in `redhat-rhmi-middleware-monitoring-operator` namespace
    > Value for `receivers[1].pagerduty_configs.service_key` should be `new_test`
16. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
17. Open **Workloads > Secrets**
18. Click on `alertmanager-application-monitoring` secret
19. **Reveal Values**
20. Make note of value for `receivers[2].webhook_configs.url`
21. Navigate to `redhat-rhmi-operator` namespace
22. Open **Workloads > Secrets**
23. Click on `redhat-rhmi-deadmanssnitch` secret
24. **Reveal Values**
    > Value for `url` key should match value noted earlier
25. **Actions > Edit Secret**
26. **Add Key/Value**
27. For **Key** use `SNITCH_URL`
28. For **Value** use `https://dms2.example.com`
29. Check again `alertmanager-application-monitoring` secret in `redhat-rhmi-middleware-monitoring-operator` namespace
    > Value for `receivers[2].webhook_configs.url` should be `https://dms2.example.com`
