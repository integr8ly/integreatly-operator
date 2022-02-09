---
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
  - name: rhoam
    environments:
      - osd-post-upgrade
estimate: 30m
tags:
  - manual-selection
---

# N03 - Verify that the minimal CIDR block is configured correctly

## Steps

This test-case requires a QE account for https://qaprodauth.cloud.redhat.com/beta/openshift/ and should be left for a member of the QE team.

1. Login to https://qaprodauth.cloud.redhat.com/beta/openshift/ using your credentials `<kerberos id>>-csqe`
2. Select the correct cluster to verify. This can be obtained from the Epic jira this ticket was selected from.
3. Find the cluster from the list of clusters and select it.
4. Select the Network tab from the available tabs.
5. The Network configuration should match the details below.

   - Machine CIDR: `10.11.128.0/23`
   - Service CIDR: `10.11.0.0/18`
   - Pod CIDR: `10.11.64.0/18`
   - Host Prefix: `23`
