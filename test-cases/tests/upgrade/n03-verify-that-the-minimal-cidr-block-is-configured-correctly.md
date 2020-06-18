---
tags:
  - happy-path
estimate: 30m
---

# N03 - Verify that the minimal CIDR block is configured correctly

## Steps

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

   ```
   oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
   ```

2. Verify that the cluster is using the following network settings

   - Machine CIDR: `10.11.128.0/23`
   - Service CIDR: `10.11.0.0/18`
   - Pod CIDR: `10.11.64.0/18`
   - Host Prefix: `23`
