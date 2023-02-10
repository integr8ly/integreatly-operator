---
estimate: 15m
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.2.0
      - 1.0.0
tags:
  - automated
---

# A28 - Verify pod priority class name is set on products

## Description

This test case should verify that the pod priority class is name is updated on RHSSO, UserSSO and 3scale.

## Steps

1. Log in to cluster console as kubeadmin

2. Confirm the rhmi cr has the field `priorityClassName` and its value is `rhoam-pod-priority`

3. Confirm each of the resources below have the field `priorityClassName` and its value is `rhoam-pod-priority`

### Deployments

| **Namespace**                             | **Name**                                  |
| ----------------------------------------- | ----------------------------------------- |
| redhat-rhoam-3scale                       | marin3r-instance                          |
| redhat-rhoam-3scale-operator              | threescale-operator-controller-manager-v2 |
| redhat-rhoam-cloud-resources-operator     | cloud-resource-operator                   |
| redhat-rhoam-customer-monitoring-operator | grafana-operator-controller-manager       |
| redhat-rhoam-customer-monitoring-operator | grafana-deployment                        |
| redhat-rhoam-marin3r                      | ratelimit                                 |
| redhat-rhoam-marin3r-operator             | marin3r-controller-webhook                |
| redhat-rhoam-marin3r-operator             | marin3r-controller-manager                |
| redhat-rhoam-rhsso-operator               | rhsso-operator                            |
| redhat-rhoam-user-sso-operator            | rhsso-operator                            |

### DeploymentConfigs

| **Namespace**       | **Name**           |
| ------------------- | ------------------ |
| redhat-rhoam-3scale | zync-que           |
| redhat-rhoam-3scale | zync-database      |
| redhat-rhoam-3scale | zync               |
| redhat-rhoam-3scale | system-sphinx      |
| redhat-rhoam-3scale | system-sidekiq     |
| redhat-rhoam-3scale | system-memcache    |
| redhat-rhoam-3scale | system-app         |
| redhat-rhoam-3scale | backend-worker     |
| redhat-rhoam-3scale | backend-listener   |
| redhat-rhoam-3scale | backend-cron       |
| redhat-rhoam-3scale | apicast-staging    |
| redhat-rhoam-3scale | apicast-production |

### StatefulSets

| **Namespace**         | **Name** |
| --------------------- | -------- |
| redhat-rhoam-rhsso    | keycloak |
| redhat-rhoam-user-sso | keycloak |
