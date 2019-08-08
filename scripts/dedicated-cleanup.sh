#!/usr/bin/env bash

echo this script leaves the htpasswd provider in place and just removed the dedicated-admin-operator. Note it leaves the namespace in place due to issues removing it
RELEASE=release-4.1
BASE_HREF=https://raw.githubusercontent.com/openshift/dedicated-admin-operator/${RELEASE}

oc delete -f ${BASE_HREF}/manifests/00-dedicated-admins-cluster.ClusterRole.yaml
oc delete -f ${BASE_HREF}/manifests/00-dedicated-admins-project.ClusterRole.yaml
oc delete -f ${BASE_HREF}/manifests/01-dedicated-admin-operator.ServiceAccount.yaml
oc delete -f ${BASE_HREF}/manifests/02-dedicated-admin-operator-admin.ClusterRoleBinding.yaml
oc delete -f ${BASE_HREF}/manifests/02-dedicated-admin-operator-cluster.ClusterRoleBinding.yaml
oc delete -f ${BASE_HREF}/manifests/02-dedicated-admin-operator-project.ClusterRoleBinding.yaml
oc delete -f ${BASE_HREF}/manifests/02-dedicated-admins-cluster.ClusterRoleBinding.yaml
oc delete -f ${BASE_HREF}/manifests/10-dedicated-admin-operator.Deployment.yaml
oc delete -f ${BASE_HREF}/manifests/10-dedicated-admin-operator.Role.yaml
oc delete -f ${BASE_HREF}/manifests/11-dedicated-admin-operator.RoleBinding.yaml