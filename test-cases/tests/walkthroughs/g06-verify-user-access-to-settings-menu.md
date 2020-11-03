---
products:
  - name: rhmi
    environments:
      - osd-fresh-install
    targets:
      - 2.8.0
estimate: 15m
---

# G06 - Verify user access to settings menu

## Steps

When logged into solution-explorer as a developer

1. the settings cog in the top right hand corner should be inactive.
2. if you try to access the settings page by adding the endpoint '/settings' to the URL, you should be directed to a warning page explaining you don't have access to the settings page.

When logged into solution-explerer as a customer-admin

1. the settings cog in the top right hand corner should be active.
2. the settings page can be accessed by adding the endpoint '/settings' to the URL.
