---
estimate: 15m
tags:
---

# G06 - Verify user role can be determined in solution-explorer

Acceptance Criteria:

When logged into solution-explorer as a developer

1. under the `All services` tab you should see a selection at the bottom called `Manage users`. Instead of a console link there should be text stating `Admins only`
2. the settings cog in the top right hand corner should be inactive.
3. if you try to access the settings page by adding the endpoint '/settings' to the URL, you should be directed to a warning page explaining you don't have access to the settings page.

When logged into solution-explerer as a customer-admin

1. under the `All services` tab you should see a selection at the bottom called `Manage users`. There should be an active link.
2. the settings cog in the top right hand corner should be active.
3. the settings page can be accessed by adding the endpoint '/settings' to the URL.
