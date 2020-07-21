---
targets:
  - 2.6.0
---

# C13 - Verify Fuse alert

## Description

More info: <https://issues.redhat.com/browse/INTLY-9047>

## Steps

1. Scale down `syndesis-db` for Fuse Online
   1. Login as kubeadmin
   2. **Workloads > Deployment Configs**
   3. Select `redhat-rhmi-fuse` project
   4. Scale `syndesis-db` down to 0 pods
2. Check that `FuseOnlinePostgresExporterDown` alert is firing
   1. **Networking > Routes**
   2. Select `redhat-rhmi-middleware-monitoring-operator` project
   3. Open route for `alertmanager-route`
      > `FuseOnlinePostgresExporterDown` should be firing
