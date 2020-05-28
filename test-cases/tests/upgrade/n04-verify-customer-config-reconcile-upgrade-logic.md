---
estimate: 2h
require:
  - N01
---

# N04 - verify customer config reconcile upgrade logic

## Prerequisites

1. Login via `oc` as a user with **cluster-admin** role (kubeadmin):

```
oc login --token=<TOKEN> --server=https://api.<CLUSTER_NAME>.s1.devshift.org:6443
```

2. Edit RHMIConfig in the `redhat-rhmi-operator` config to prevent the available upgrade from being applied

```
oc edit RHMIConfig rhmi-config -n redhat-rhmi-operator
```

3. Make sure that the following are as follows (change if required):

- `spec.upgrade.alwaysImmediately` set to false
- `spec.upgrade.duringNextMaintenance` set to false
- `spec.maintenance.applyOn` set to ""

4. Save your changes. The available upgrade should not be applied with the settings described above

5. Edit RHMIConfig in the `redhat-rhmi-operator` config again to apply the upgrade

```
oc edit RHMIConfig rhmi-config -n redhat-rhmi-operator
```

6. Make sure that the following are as follows (change if required):

- `spec.upgrade.alwaysImmediately` set to true

7. Verify that after changing the config as described above the upgrade was eventually applied automatically
