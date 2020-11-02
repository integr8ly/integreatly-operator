---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
estimate: 15m
---

# M05B - Verify secrets

More info:

- <https://issues.redhat.com/browse/INTLY-4885>
- <https://issues.redhat.com/browse/INTLY-9096>

## Steps

### Verify pagerduty secret

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-managed-api-middleware-monitoring-operator` namespace
3. Open **Workloads > Secrets**
4. Click on `alertmanager-application-monitoring` secret
5. **Reveal Values**
6. Make note of value for `receivers[1].pagerduty_configs.service_key`
7. Navigate to `redhat-managed-api-operator` namespace
8. Open **Workloads > Secrets**
9. Click on `redhat-managed-api-pagerduty` secret
10. **Reveal Values**
    > Value for `serviceKey` key should match value noted earlier
11. **Actions > Edit Secret**
12. **Add Key/Value**
13. For **Key** use `PAGERDUTY_KEY`
14. For **Value** use `new_test`
15. Check again `alertmanager-application-monitoring` secret in `redhat-managed-api-middleware-monitoring-operator` namespace
    > Value for `receivers[1].pagerduty_configs.service_key` should be `new_test`

### Verify deadmanssnitch secret

1. Navigate to `redhat-managed-api-middleware-monitoring-operator` namespace
2. Open **Workloads > Secrets**
3. Click on `alertmanager-application-monitoring` secret
4. **Reveal Values**
5. Make note of value for `receivers[2].webhook_configs.url`
6. Navigate to `redhat-managed-api-operator` namespace
7. Open **Workloads > Secrets**
8. Click on `redhat-managed-api-deadmanssnitch` secret
9. **Reveal Values**
   > Value for `url` key should match value noted earlier
10. **Actions > Edit Secret**
11. **Add Key/Value**
12. For **Key** use `SNITCH_URL`
13. For **Value** use `https://dms2.example.com`
14. Check again `alertmanager-application-monitoring` secret in `redhat-managed-api-middleware-monitoring-operator` namespace
    > Value for `receivers[2].webhook_configs.url` should be `https://dms2.example.com`
