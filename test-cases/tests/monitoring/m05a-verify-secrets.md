---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.6.0
estimate: 15m
---

# M05A - Verify secrets

More info:

- <https://issues.redhat.com/browse/INTLY-4885>
- <https://issues.redhat.com/browse/INTLY-9096>

## Steps

### Verify pagerduty secret

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
11. Compare that the `redhat-rhmi-pagerduty` secret matches the value for `receivers[1].pagerduty_configs.service_key` in the `alertmanager-application-monitoring` secret in `redhat-rhmi-middleware-monitoring-operator` namespace

### Verify deadmanssnitch secret

1. Navigate to `redhat-rhmi-middleware-monitoring-operator` namespace
2. Open **Workloads > Secrets**
3. Click on `alertmanager-application-monitoring` secret
4. **Reveal Values**
5. Make note of value for `receivers[2].webhook_configs.url`
6. Navigate to `redhat-rhmi-operator` namespace
7. Open **Workloads > Secrets**
8. Click on `redhat-rhmi-deadmanssnitch` secret
9. **Reveal Values**
10. Compare that the `redhat-rhmi-deadmanssnitch` secret matches the value for `receivers[2].webhook_configs.url` in the `alertmanager-application-monitoring` secret in `redhat-rhmi-middleware-monitoring-operator` namespace
