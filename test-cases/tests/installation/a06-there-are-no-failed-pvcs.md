---
tags:
  - happy-path
estimate: 15m
---

# A06 - There are no failed PVCs

## Prerequisites

Login as **kube-admin** or user with **admin** role

## Steps

1.  Select **Administrator** from the left side panel
2.  Go to Storage and select **Persistent Volume Claims**
3.  Select Project as **all projects** in the drop down menu on the top of the page and verify that there are no failed PVCs
    > All PVCs should have **bound** status
    >
    > 1. redhat-rhmi-fuse
    >
    >    - syndesis-meta
    >    - syndesis-prometheus
    >
    > 2. redhat-rhmi-middleware-monitoring-operator
    >
    >    - prometheus-application-monitoring-db-prometheus-application-monitoring-0
    >
    > 3. redhat-rhmi-solution-explorer
    >
    >    - user-walkthroughs
    >
    > 4. redhat-rhmi-operator
    >
    >    - standard-authservice-postgresql
