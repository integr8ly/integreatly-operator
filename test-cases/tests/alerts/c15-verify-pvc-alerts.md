---
environments:
  - osd-post-upgrade
estimate: 30m
targets:
  - 2.6.0
---

# C15 - Verify PVC alerts

## Description

More info: <https://issues.redhat.com/browse/INTLY-9432>

## Steps

1. Open a shell in a pod that has a PV attached
2. Use fallocate to make a file big enough to bring it past 97% full, but maybe less than 100% so that you don't cause other problems
3. See that the alerts fire
4. Delete your fallocated file to clean up

E.g. for the fuse-prometheus pod in the redhat-rhmi-fuse namespace:

Try to figure out how much space you'll need to get to 98%

```bash
sh-4.2$ df -h /prometheus/
Filesystem      Size  Used Avail Use% Mounted on
/dev/nvme3n1    9.8G  293M  9.5G   3% /prometheus
```

Might take a little trial and error to get into the right range:

```bash
sh-4.2$ fallocate -l 9250M /prometheus/big-test-file
sh-4.2$ df -h /prometheus/
Filesystem      Size  Used Avail Use% Mounted on
/dev/nvme3n1                          9.8G  9.6G  207M  98% /prometheus
```

...check prometheus to see the alert go to pending (might take a short period of time)...

Clean up file:

```bash
sh-4.2$ rm /prometheus/big-test-file
sh-4.2$ df -h /prometheus/
Filesystem      Size  Used Avail Use% Mounted on
/dev/nvme3n1    9.8G  293M  9.5G   3% /prometheus
```
