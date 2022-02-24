---
automation:
  - MGDAPI-3454
components:
  - product-sso
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
    targets:
      - 1.13.0
      - 1.16.0
estimate: 15m
tags:
  - automated
---

# B09 - Verify dedicated admin users are synced with User SSO

## Prerequisites

- oc (OpenShift client) cluster admin access (user logged in as kubeadmin)
- Dedicated Admin access to a cluster console (customer-admin user)

## Steps

1. As a kubeadmin, create a user which will be used for testing OpenShift user -> User SSO synchronization

```bash
cat << EOF | oc create -f -
---
apiVersion: user.openshift.io/v1
kind: User
metadata:
  name: test-user99
EOF
```

2. As a customer-admin, log in to OpenShift console and then log in to User SSO
3. Create the testing user in the Keycloak console

```
Username: test-user99
First name: Test
Last name: User 99
User enabled: true
Email Verified: true
```

4. Promote the user in Openshift by adding it to the `users` list of the `dedicated-admins` group

```bash
# Edit dedicated-admins group by adding test-user99 to the list of users in the group
oc edit group dedicated-admins
```

5. Wait for RHOAM to reconcile the `KeycloakUser` CR in the `redhat-rhoam-user-sso` namespace

```bash
oc get keycloakuser $(oc get keycloakuser -n redhat-rhoam-user-sso | grep test-user99 | awk '{print $1}') -n redhat-rhoam-user-sso -o yaml
```

> Ensure that status is successful
> Ensure that the user has the following groups
>
> - dedicated-admins
> - dedicated-admins/realm-managers

6. Go to User SSO keycloak console, in the list of users select `test-user99`
   > Verify that the user is now a member of `/dedicated-admins`, `/dedicated-admins/realm-managers`, and `/rhmi-developers`
