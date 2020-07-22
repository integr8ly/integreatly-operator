---
targets:
  - 2.6.0
---

# N09 - Verify that upgrades rollout can be paused

## Description

We need to  have a way of pausing upgrades to prevent upgrade of the whole fleet in a scenario where we discover a critical regression in latest RHMI version, after it has been promoted to production environment.

## Prerequisites

- pre-upgrade cluster
- VPN connection
- familiarity with managed tenants repo and release process

## Steps

1. Change RHMIConfig on a pre-upgrade cluster to postpone upgrades for later.
2. Publish new version into the managed tenants repo(MT). You will need cooperation with release coordinator for this step.
3. Verify that InstallPlan for rhmi-operator was created but not apprroved. This could take up to 30mins.
4. Verify that RHMIConfig status shows scheduled upgrade
5. Follow this SOP in "stage" environment: https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/cssre_info/info_cssre_stopping_version_rollout.md 
6. Verify that InstallPlan for the new version was deleted and RHMIConfig status no longer shows scheduled upgrade. This could take up to 30mins.