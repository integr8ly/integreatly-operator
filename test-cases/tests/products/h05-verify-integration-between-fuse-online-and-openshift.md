---
automation:
  - INTLY-7433
components:
  - product-fuse
environments:
  - osd-post-upgrade
estimate: 30m
targets:
  - 2.6.0
---

# H05 - Verify integration between Fuse Online and OpenShift

## Prerequisites

Login as a user in the **developer** group.

## Steps

1. Open the Fuse Online Console and create a new Integration

   ```
   Note: a) Use Fuse instance that is linked in Solution
            Explorer
         b) Namespace of the deployment-config may vary based
            on the namespace of Fuse instance
   ```

   1. Select API Provider
   2. Choose `Create a new OpenAPI 3.x document`, and Next
   3. Add a new Path `/dummy`
   4. Add a new Get Operation
   5. Add a Response to the Operation `200 OK`
   6. Write something in the Response Description
   7. Click on Save, then Next, and Publish
   8. Assign a name to the integration, and Save and Publish

   > The Integration should be created and running in Fuse
   >
   > The deploymentconfig should be created in OpenShift:
   >
   > - i-`integration-name`
   >
   > ```
   > oc get deploymentconfigs --namespace=<fuse-namespace> | grep <integration-name>
   > ```

2. Delete the Integration
   > The integration should be deleted from Fuse and the deploymentconfig from OpenShift
