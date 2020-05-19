---
estimate: 1h
---

# J02 - Verify all AWS snapshot succeed

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

You will need to be logged into the appropriate AWS console.

Acceptance Criteria:

All backups jobs should succeed

1. Postgres

   1. Codeready
   2. Managed Fuse # _Currently waiting on Fuse release_
   3. Application (User) SSO # _Currently waiting on RHSSO release_
   4. Cluster SSO
   5. UPS
   6. 3scale

2. Redis
   1. 3scale x 2

Steps:

1. To creating the snapshots run this [script](https://gist.github.com/ciaranRoche/d98131d81b8150eb323215469d48bcb1)
2. Follow the output from the script and run the commands shown.
   1. This will verify the snapshots have been created
   2. Also to verify automated snapshots and backups are in place for both redis and postgres
