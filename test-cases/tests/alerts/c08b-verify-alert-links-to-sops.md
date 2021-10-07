---
estimate: 15m
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.3.0
      - 1.6.0
      - 1.9.0
      - 1.12.0
---

# C08B - Verify alert links to SOPs

## Description

This test case should verify that all SOP links in alerts point to correct SOPs.

## Steps

### Check that all SOPs for URLs in `sop_url` exist

1. Open OpenShift console in your browser
2. Login as admin
3. Find route for Prometheus in `redhat-rhoam-observability` namespace
4. Open its URL
5. Change URL path to `/api/v1/rules`
6. Download json from the `/api/v1/rules` API endpoint from previous step
7. List all `sop_url`s in the json with `cat rules.json | jq '[.data.groups[].rules[].annotations.sop_url] | unique'`
8. By opening the URLs check that all SOPs exist
