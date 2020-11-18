---
estimate: 15m
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 0.2.0
      - 1.0.0
---

# A28 - Verify pod priority class name is set on products

## Description

This test case should verify that the pod priority class is name is updated on RHSSO, UserSSO and 3scale.

## Steps

1. Log in to cluster console as kubeadmin

2. Confirm rhoam CR has the field `priorityClassName` and it's value is `managed-service-priority`

3. Confirm RHSSO and USERSSO `keycloak` statefulsets have the field `priorityClassName` with the value of `managed-service-priority`

4. Confirm threescale deployment configs for the below deployments have the field `priorityClassName` with the value of `managed-service-priority`

````"apicast-production",
   	"apicast-staging",
   	"backend-cron",
   	"backend-listener",
   	"backend-worker",
   	"system-app",
   	"system-memcache",
   	"system-sidekiq",
   	"system-sphinx",
   	"zync",
   	"zync-database",
   	"zync-que",```
````
