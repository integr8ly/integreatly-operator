---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.2.0
      - 1.5.0
      - 1.6.0
      - 1.8.0
      - 1.11.0
      - 1.14.0
      - 1.20.0
      - 1.23.0
      - 1.26.0
      - 1.29.0
      - 1.32.0
      - 1.35.0
      - 1.38.0
      - 1.41.0
estimate: 1h
tags:
  - destructive
---

# H25 - Verify rate limiting can be disabled and re-enabled by following the SOP

## Description

> Note: double-check that workload webapp is not already deployed before attempting to deploy it by checking if `workload-web-app` namespace exists in the cluster.

This test case should prove that it is possible for SRE to disable/enable rate limiting service without affecting the RHOAM services availability

## Prerequisites

- access to `cloud-services-qe-reporting@redhat.com` mailing list (optional)
  - you can monitor the alerts directly in the Observability Prometheus instance instead. Open it before starting with the SOP and check continuously
- [workload webapp](https://github.com/integr8ly/workload-web-app) should be running on the cluster https://github.com/integr8ly/workload-web-app/

## Steps

1. Go to https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/rate-limit/disable.md
2. Follow and validate the steps in SOP for disabling rate limit service
3. Open the RHOAM Grafana Console in the `redhat-rhoam-customer-monitoring` namespace

```bash
open "https://$(oc get route grafana-route -n redhat-rhoam-customer-monitoring -o=jsonpath='{.spec.host}')"
```

4. Select the **Workload App** dashboard
   > Validate that requests to 3scale application are not failing after rate limiting service was disabled
   > Note: Downtime of up to 5 minutes is acceptable as per the service definition
5. Search for alerts in `cloud-services-qe-reporting@redhat.com` mailing list
   > Make sure no critical alert is firing (you might see some alerts with severity "warning")
6. Follow and validate the steps in SOP for re-enabling rate limit service
7. Go back to **Workload App** dashboard
   > Validate that requests to 3scale application are not failing after rate limiting service was enabled again
   > Note: Downtime of up to 5 minutes is acceptable as per the service definition
8. Open the RHOAM Grafana Console in the `redhat-rhoam-customer-monitoring` namespace

```bash
open "https://$(oc get route grafana-route -n redhat-rhoam-customer-monitoring -o=jsonpath='{.spec.host}')"
```

9. Validate that the requests made by workload-web-app are displaying in the graphs
10. Search for alerts in `cloud-services-qe-reporting@redhat.com` mailing list (if not checked via Prometheus directly)
    > Make sure no critical alert is firing (you might see some alerts with severity "warning")
