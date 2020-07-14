---
estimate: 30m
components:
  - product-codeready
  - openshift
targets: []
automation_jiras:
  - INTLY-7434
---

# H06 - Verify integration between CodeReady and OpenShift

## Prerequisites

Login as a user in the **rhmi-developer** group.

### rhmi-developer group

Test users should be automatically added to this group after logging in for the first time.

Membership of the **rhmi-developer** can be verified by executing `oc get group rhmi-developers` as kubeadmin.

## Steps

1. Open the CodeReady Console
2. Create a new Go Example (Stack: Go; Project: example) Workspace
3. Verify the workspace is available in the CodeReady console.
4. Verify the pods for the workspace have been created successfully and are running.
   > Two new pods should be created in OpenShift, ex: workspaceo02g6g0r8aovuntk.che-jwtproxy; workspaceo02g6g0r8aovuntk.go-cli
   >
   > Run the command below as customer-admin to check
   >
   > ```
   > oc get pods --namespace=redhat-rhmi-codeready-workspaces | grep workspace
   > ```
5. Open the Workspace, select Terminal, Run Task, and execute `test outyet` (The `test outyet` task is part of the Go Example project)
   > One single test would be executed and it should PASS
6. Stop and Delete the Workspace
   > The workspace should be deleted from CodeReady and the pods from OpenShift
