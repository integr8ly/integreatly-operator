---
estimate: 15m
tags:
  - happy-path
---

# H09 - Verify RHMI ALERT metrics are scraped by openshift-monitoring

## Acceptance Criteria

Verify RHMI ALERT metrics are scraped by openshift-monitoring. https://issues.redhat.com/browse/INTLY-6610. Reach out to Mark Freer/ Alan Moran if more details are required.

## Prerequisites

Login to OpenShift console as a **kubeadmin** (user with cluster-admin permissions).

## Steps

1. Verify a service monitor exists to federate RHMI ALERT metric
2. Verify new namespace where the service monitor will reside (Has openshift.io/cluster-monitoring: "true" label)
3. Verify prometheus-k8s has view access to redhat-rhmi-middleware-monitoring-operator (for federate access)

- Go to cluster
- routes --> get prometheus route
- Log in
- Status --> targets
- Check for `*federate*`

4. Verify RHMI alerts appear in cluster prometheus
