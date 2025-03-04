---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.32.0
      - 1.35.0
      - 1.38.0
      - 1.39.0
      - 1.42.0
estimate: 5m
---

# C08B - Verify alert links to SOPs

## Description

Automated, see [verify_alert_links_to_SOPs.go](https://github.com/integr8ly/integreatly-operator/blob/master/test/common/verify_alert_links_to_SOPs.go). The `GITLAB_TOKEN` required for the test is passed into test container, see [runFunctionalTest.groovy](https://gitlab.cee.redhat.com/integreatly-qe/ci-cd/-/blob/master/vars/runFunctionalTest.groovy#L83).

The trouble is that currently the tests are executed in OSD under test in pod in container so it is not behind VPN thus CEE GitLab where SOPs are stored is not accessible.

This test case should verify that all SOP links in alerts point to correct SOPs.

## Prerequisites

Ability to decrypt the secrets in CEE GitLab integreatly-qe/ci-cd repository (or ask any QE member). Alternatively generate a [Personal Access Token](https://gitlab.cee.redhat.com/-/profile/personal_access_tokens).

## Steps

### Check that all SOPs for URLs in `sop_url` exist

1. `oc login...` into the cluster to be tested
2. Get the GitLab API token from ci-cd repo in CEE GitLab: `cat ci-cd/secrets/gitlab-api-token`
   > Or use your own Personal Access Token, the key is to have access to [integreatly-help](https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help)
3. Navigate to integreatly-operator repository
4. Make sure your machine is behind VPN
5. Run the C18B test

`GITLAB_TOKEN=<gitlab-api-token> LOCAL=false INSTALLATION_TYPE=managed-api TEST=C08B make test/e2e/single`
