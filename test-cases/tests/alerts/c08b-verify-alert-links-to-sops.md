---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.28.0
estimate: 5m
---

# C08B - Verify alert links to SOPs

## Description

This test case should verify that all SOP links in alerts point to correct SOPs.

## Prerequisites

- Ability to decrypt the secrets in CEE GitLab integreatly-qe/ci-cd repository (or ask any QE member)

## Steps

### Check that all SOPs for URLs in `sop_url` exist

1. `oc login...` into the cluster to be tested
2. Get the GitLab API token from ci-cd repo in CEE GitLab: `cat ci-cd/secrets/gitlab-api-token`
3. Navigate to integreatly-operator repository
4. Make sure your machine is behind VPN
5. Run the C18B test

`GITLAB_TOKEN=<gitlab-api-token> LOCAL=false INSTALLATION_TYPE=managed-api TEST=C08B make test/e2e/single`
