---
automation:
  - MGDAPI-1260
products:
  - name: rhoam
    environments:
      - osd-fresh-install
      - osd-post-upgrade
    targets:
      - 1.2.0
      - 1.6.0
      - 1.9.0
tags:
  - automated
  - destructive
estimate: 15m
---

# C19 - Validate creation of invalid username triggers alert

https://github.com/integr8ly/integreatly-operator/blob/master/test/common/alerts_invalid_username.go
