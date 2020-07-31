---
environments:
  - osd-post-upgrade
targets:
  - 2.3.0
  - 2.8.0
---

# H12 - Verify Syndesis Prometheus PV size

## Description

Verify that PV size for syndesis-prometheus was increased to 10Gb. This should be verified for both upgrade and fresh installation of RHMI.

More info: <https://issues.redhat.com/browse/INTLY-7188>

## Steps

1. Open OpenShift console in your browser
2. Login as admin
3. Open `redhat-rhmi-fuse` project
4. Open **Workloads -> Pods**
5. Open `syndesis-prometheus` pod
6. In **Volumes** section open `syndesis-prometheus` PVC
7. Check that capacity is 10Gi
