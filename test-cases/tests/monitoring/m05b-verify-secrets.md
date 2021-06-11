---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
estimate: 15m
tags:
  - manual-selection
---

# M05B - Verify secrets

More info:

- <https://issues.redhat.com/browse/INTLY-4885>
- <https://issues.redhat.com/browse/INTLY-9096>

Obsolete - secrets are managed by 3rd party (Hive, OCM) now.

## Steps

### Verify pagerduty secret

1. Login into OpenShift console as kubeadmin
2. Navigate to `redhat-rhoam-middleware-monitoring-operator` namespace
3. Open **Workloads > Secrets**
4. Click on `alertmanager-application-monitoring` secret
5. **Reveal Values**
6. Make note of value for `receivers[1].pagerduty_configs.service_key`
7. Navigate to `redhat-rhoam-operator` namespace
8. Open **Workloads > Secrets**
9. Click on `redhat-rhoam-pagerduty` secret
10. **Reveal Values**
11. Compare that the `redhat-rhoam-pagerduty` secret matches the value for `receivers[1].pagerduty_configs.service_key` in the `alertmanager-application-monitoring` secret in `redhat-rhoam-middleware-monitoring-operator` namespace

### Verify deadmanssnitch secret

1. Navigate to `redhat-rhoam-middleware-monitoring-operator` namespace
2. Open **Workloads > Secrets**
3. Click on `alertmanager-application-monitoring` secret
4. **Reveal Values**
5. Make note of value for `receivers[2].webhook_configs.url`
6. Navigate to `redhat-rhoam-operator` namespace
7. Open **Workloads > Secrets**
8. Click on `redhat-rhoam-deadmanssnitch` secret
9. **Reveal Values**
10. Compare that the `redhat-rhoam-deadmanssnitch` secret matches the value for `receivers[2].webhook_configs.url` in the `alertmanager-application-monitoring` secret in `redhat-rhoam-middleware-monitoring-operator` namespace
