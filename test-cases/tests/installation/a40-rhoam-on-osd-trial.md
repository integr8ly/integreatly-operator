---
products:
  - name: rhoam
    environments:
      - external
estimate: 2h
tags:
  - per-release
---

# A40 - RHOAM on OSD Trial

## Description

Verify RHOAM installation on OSD Trial works as expected.

## Steps

1. Provision an OSD Trial Cluster through [Jenkins](https://master-jenkins-csb-intly.apps.ocp4.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow/)
   - Reach out for AWS credentials needed to provision an OSD Trial Cluster
   - Check `osdTrial` checkbox and for phases choose `provisionCluster` and `installProduct`
2. After pipeline finishes log into the cluster using `oc` and the provided kubeadmin credentials
3. Verify RHOAM installation completed successfully, and uses the correct `Evaluation` (`0`) quota config

```
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status'
```

4. Go to OCM UI, select your OSD Trial cluster and select "upgrade", then select "using quota"
5. Your OSD cluster should now have the type "OSD" (previously it was OSD Trial)
6. Select your cluster, go to Addons -> RHOAM -> Configuration
7. Try to change the "Quota" param and click on "Update"
8. After a while, .toQuota field should be updated to the value you've selected

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.toQuota'
```

9. Once the new quota has been applied to the RHOAM cluster, `.quota` field should be updated

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
```

10. Trigger uninstall of the addon via the Cluster OCM UI
11. Verify uninstall completes successfully
