---
estimate: 15m
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.3.0
      - 2.4.0
      - 2.6.0
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
---

# C08 - Verify alert links to SOPs

## Description

This test case should verify that all SOP links in alerts point to correct SOPs, that links to scaling_plans were removed from alerts and that scaling plans are moved to SOPs or referenced from SOPs.

More info: <https://issues.redhat.com/browse/INTLY-7745>

## Steps

### Check that all `scaling_plan` links were removed from alerts

1. Open OpenShift console in your browser
2. Login as admin
3. Find route for Prometheus in `redhat-rhmi-middleware-monitoring-operator` namespace
4. Open its URL
5. Change URL path to `/api/v1/rules`
6. Search for `scaling_plan`
   > There should not be any `scaling_plan`s

### Check that all SOPs for URLs in `sop_url` exist

1. Download json from the `/api/v1/rules` API endpoint from previous step
2. List all `sop_url`s in the json with `cat rules.json | jq '[.data.groups[].rules[].annotations.sop_url] | unique'`
3. By opening the URLs check that all SOPs exist
