---
automation:
  - INTLY-7435
components:
  - product-3scale
environments:
  - osd-post-upgrade
estimate: 1h
targets:
  - 2.8.0
---

# H07 - Verify 3scale User Promotion Works and Stays Permanent

## Prerequisites

Login to OpenShift console as a **kubeadmin** (user with cluster-admin permissions).
In different window, login as a user from the **developer** group.
In another window, login as a user from the **dedicated-admin** group.

## Steps

1. As a **dedicated-admin** user, go to Projects -> redhat-rhmi-3scale
2. Select Networking -> Routes
3. Go to 3scale API Management console (route is starting with "https://3scale-admin")
4. Select "Authenticate through Red Hat Single Sign-On" and login as a user with **dedicated-admin** permissions
5. Select Account Settings (Top right corner) and select Users -> Listing
6. Select a user from the **developer** group (the one you've previously logged in)
7. Under Administrative, select Admin and hit "Update User"
   > The **developer** user you've changed should have "admin" role now (see the users table),
8. Try to login to 3scale API Management console as a **developer** user you previously promoted
   > You should successfully log in and have "admin" permissions: go to Account Settings (Top right corner) and you should be able to see "Users" menu in left column)
9. Force reconcile rhmi-operator (by scaling it down & up): As a **kubeadmin**, go to Projects -> redhat-rhmi-operator -> Workloads -> Deployments -> rhmi-operator and hit the "down arrow" and "up arrow" to scale it down and up
10. Try to login again to 3scale API Management console as a user from **developer** group
    > You should still be able to log in and still have "admin" permissions
