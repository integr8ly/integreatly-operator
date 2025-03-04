---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 0.1.0
      - 0.2.0
      - 1.0.0
      - 1.5.0
      - 1.8.0
      - 1.11.0
      - 1.14.0
      - 1.18.0
      - 1.21.0
      - 1.24.0
      - 1.27.0
      - 1.30.0
      - 1.33.0
      - 1.36.0
      - 1.39.0
estimate: 15m
tags:
  - destructive
  - manual-selection
---

# C14B - Verify 3scale UIBBT alerts

## Description

Test was automated. Test will verify if 3scale UIBBT alerts are firing

## Steps

1. `oc login` into the cluster
2. Navigate to integreatly-operator repository
3. Run the test `INSTALLATION_TYPE=managed-api DESTRUCTIVE=true LOCAL=false TEST=C14B make test/e2e/single`
