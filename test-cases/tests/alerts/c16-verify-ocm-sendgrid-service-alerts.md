---
environments:
  - osd-post-upgrade
estimate: 30m
tags:
  - per-release
---

# C16 - Verify OCM SendGrid Service alerts

## Description

This test should verify that the expected alerts for the OCM SendGrid Service exist in the OCM Prometheus instance.

## Steps

### Stage

1. Go to the [OCM staging Prometheus instance](https://prometheus.app-sre-stage-01.devshift.net/alerts)

2. Login via the OpenShift option, using the `github-app-sre` identity provider

   - If you do not have permissions via this identity provider, contact the `sd-app-sre` Slack channel

3. Ensure each alert defined in [the alert definition](https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/resources/observability/prometheusrules/ocm-sendgrid-svc-stage.prometheusrules.yaml) exists in the list of Prometheus alerts, and none are firing

### Production

1. Go to the [OCM production Prometheus instance](https://prometheus.app-sre-prod-04.devshift.net/alerts)

2. Login via the OpenShift option, using the `github-app-sre` identity provider

   - If you do not have permissions via this identity provider, contact the `sd-app-sre` Slack channel

3. Ensure each alert defined in [the alert definition](https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/resources/observability/prometheusrules/ocm-sendgrid-svc-production.prometheusrules.yaml) exists in the list of Prometheus alerts, and none are firing
