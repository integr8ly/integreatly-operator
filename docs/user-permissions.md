# Integreatly Users and Permissions on OSD

Within an RHMI cluster there are two main groups of users that the operator interacts with these are: 

- Dedicated Admins Group
- RHMI Developers Group

### Dedicated Admins Group
The [dedicated admins group](https://docs.openshift.com/dedicated/4/administering_a_cluster/dedicated-admin-role.html) exists in OSD (OpenShift Dedicated) clusters by default and has a number of associated RBAC permissions that grant
members in this group a higher level of priveleges to allow them to effectively administrate their cluster. The integreatly operator leverages this
group on OSD to decide which users should become admins in the managed services on cluster. 

#### Additional permissions granted to members of the dedicated-admins group

**3scale API Managment**

-  Added as an admin user in 3scale that can do all the things a 3scale admin can do such as mange other [users permissions within 3scale](https://access.redhat.com/documentation/en-us/red_hat_3scale_api_management/2.8/html/admin_portal_guide/inviting-users-managing-rights)
-  Allowed edit access to 3scale routes. This allows deciding which router the admin wants a route to be managed by. For example in a private cluster
you may want some routes to land on the public router

**User RHSSO**

-  Added to the Master realm, granted permissions to manage users in the master realm and is given the ability to create and fully manage other realms

**Code Ready Workspaces**

- Given the permission to manage the GH client secret that allows CRW to interact with you GH account.

**AMQ Online**

- Given access to create new [AMQ Online configuration files](https://access.redhat.com/documentation/en-us/red_hat_amq/7.6/html/installing_and_managing_amq_online_on_openshift/configuring-messaging).


#### Membership

Membership of the dedicated admins group is managed cloud.redhat.com. Once a user is added to this group, the integreatly operator will ensure that users in this group are granted the correct roles and permissions in the managed services.

### RHMI Developer Group

The RHMI developer group is added and manged by the operator. It represents the average user on an RHMI developer group and allows us to specify RBAC rules in one place that
will affect all users. Some users are excluded from the RHMI group these include SRE users and service accounts.

#### Additional permissions granted to members of the RHMI Developer group

**Fuse Online**

- Allowed view access to the fuse online namespace and to pod logs within that namespace. This is to allow developers to debug the Fuse Online integration pods.

**User RHSSO**

- Members of this group are allowed to login to User SSO. However they will have no permissions until granted addintion permissions by a member of the dedicated-admin group

#### Membership

Membership of the RHMI Developers group is automatically given to any user that has successfully authenticated and exists as a user with OpenShift. The only exceptions to this rule are members of the SRE team and service accounts. 
