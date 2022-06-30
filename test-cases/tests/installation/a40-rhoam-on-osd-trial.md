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

1. Login to [OCM UI (staging environment)](https://qaprodauth.cloud.redhat.com/beta/openshift/)
2. Get the [OCM API Token](https://qaprodauth.cloud.redhat.com/beta/openshift/token)
3. Provision an OSD Trial Cluster through [Jenkins](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/ManagedAPI/job/managed-api-install-addon-flow/)
   - Specify your ocmAccessToken in the pipeline parameters.
   - Reach out for AWS credentials needed to provision an OSD Trial Cluster
   - Check `osdTrial` checkbox and for phases choose `provisionCluster` and `installProduct`
4. After pipeline finishes log into the cluster using `oc` and the provided kubeadmin credentials
5. Verify RHOAM installation completed successfully, and uses the correct `Evaluation` (`0`) quota config

```
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status'
```

6.  Login to OCM with the token provided.

```bash
ocm login --url=https://api.stage.openshift.com/ --token=<YOUR_TOKEN>
```

7.  Upgrade your trial cluster "using quota"

```bash
ocm patch /api/clusters_mgmt/v1/clusters/$CLUSTER_ID --body=<<EOF
{
   "billing_model":"standard",
   "product":{
      "id":"osd"
   }
}
EOF
```

8. Your OSD cluster should now have the `red-hat-clustertype:OSD`.

```bash
ocm get /api/clusters_mgmt/v1/clusters/$CLUSTER_ID | jq '.aws.tags'
```

9. Set Quota value

```bash
QUOTA_VALUE=<QUOTA_VALUE>
```

10. Change Quota value

```bash
ocm patch /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service --body=<<EOF
{
   "parameters":{
      "items":[
         {
            "id":"addon-managed-api-service",
            "value":"$QUOTA_VALUE"
         }
      ]
   }
}
EOF
```

11. After a while, .toQuota field should be updated to the value you've selected

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.toQuota'
```

11. Once the new quota has been applied to the RHOAM cluster, `.quota` field should be updated

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq -r '.status.quota'
```

12. Trigger uninstall of the addon via the Cluster

```bash
ocm delete /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons/managed-api-service
```

13. Verify uninstall completes successfully. Once the RHOAM CR is deleted the addon installation with id=managed-api-service shouldn't be listed.

```bash
ocm get /api/clusters_mgmt/v1/clusters/$CLUSTER_ID/addons
```

Additionally,during the uninstallation you can monitor the RHMI CR stage or log into the openshift console and ensure all the redhat-rhoam namespaces have been deleted.

```bash
oc get rhmi rhoam -n redhat-rhoam-operator -o json | jq .status.stage
```
