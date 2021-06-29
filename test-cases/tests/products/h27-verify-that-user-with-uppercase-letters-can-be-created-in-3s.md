---
products:
  - name: rhoam
    environments:
      - osd-fresh-install
    targets:
      - 1.4.0
      - 1.5.0
      - 1.8.0
estimate: 30m
---

# H27 - Verify that user with uppercase letters can be created in 3scale

## Description

This test verifies that if there is an existing user with uppercase letters in the name in OpenShift, that user can be also created in 3scale (during RHOAM installation). The username in 3scale should be with lowercase letters

## Prerequisites

- OSD cluster with RHOAM installed
- Kubeadmin access to the OSD cluster
- Admin access to some github organization
- Github user with at least one uppercase letter in the username

## Steps

**Set up Github IDP for OSD cluster**

1. Go to https://qaprodauth.cloud.redhat.com/beta/openshift/ -> select your cluster -> Access control -> Add identity providers
2. Fill in the details (Client ID, Client Secret, add the organization you are a member of and have admin access to it)
3. Log in to your cluster via Github IDP (go to OpenShift console, select Github IDP)
4. Verify that the user you've logged in with has an uppercase letters in its name

```bash
oc get users | awk '{print $1}' | grep -i <your-username>
```

5. In OpenShift console (when logged in as your github user), select the launcher on the top right menu -> API Management -> Github IDP and log in to 3scale
   > Verify that you can successfully log in
6. Go to Account settings (top right menu) -> Personal -> Personal Details
   > Verify that your username contains only lowercase letters
7. Change some letter in your username to uppercase letter (e.g. myuser -> Myuser) and confirm the change
   > Verify that after RHOAM operator reconciles (~5 minutes), your username is changed back to lowercase letters
