---
environments:
  - osd-fresh-install
estimate: 15m
targets:
  - 2.6.0
---

# G16 - Verify settings notifications pop up

## Steps

Log into solution explorer in a private browser session as a user from the dedicated-admin group for example customer-admin01.

1. Select the settings cog in the top right hand corner, it should be active.
2. The top of the settings page should have a green dismissable banner at the top.
3. Close the banner.
4. Refresh the page and confirm the banner does not reappear.
5. Close the private browser session.
6. Open another private browser session and log into Solution Explorer as the same user.
7. Return to the Solution Explorer settings page.
8. Confirm the banner is again visible at the top of the settings page.
