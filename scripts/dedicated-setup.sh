#!/usr/bin/env bash



PASSWORD=$(openssl rand -base64 12)
RELEASE=release-4.1
BASE_HREF=https://raw.githubusercontent.com/openshift/dedicated-admin-operator/${RELEASE}

echo Targeting $(oc whoami --show-server)
sleep 1


echo Setting up htpasswd IDP

if [[ ! -f "htpasswd" ]]; then
  echo creating htpasswd file
  touch htpasswd
fi

htpasswd -b htpasswd customer-admin ${PASSWORD}
htpasswd -b htpasswd cluster-admin ${PASSWORD}

echo user added customer-admin ${PASSWORD}
oc delete secret htpasswd-secret -n openshift-config
oc create secret generic htpasswd-secret --from-file=htpasswd=htpasswd -n openshift-config
oc patch oauth cluster --type=merge -p '{ "spec": { "identityProviders": [{ "name": "Test Identity Provider", "challenge": true, "login": true, "mappingMethod": "claim", "type": "HTPasswd", "htpasswd": { "fileData": { "name": "htpasswd-secret" } } }] } }'

echo creating dedicated-admins group and adding users
oc adm groups new dedicated-admins
oc adm groups add-users dedicated-admins customer-admin

echo Installing dedicated admin operator from release ${RELEASE}

oc apply -f ${BASE_HREF}/manifests/00-dedicated-admins-cluster.ClusterRole.yaml
oc apply -f ${BASE_HREF}/manifests/00-dedicated-admins-project.ClusterRole.yaml
oc apply -f ${BASE_HREF}/manifests/00-openshift-dedicated-admin.Namespace.yaml
oc apply -f ${BASE_HREF}/manifests/01-dedicated-admin-operator.ServiceAccount.yaml
oc apply -f ${BASE_HREF}/manifests/02-dedicated-admin-operator-admin.ClusterRoleBinding.yaml
oc apply -f ${BASE_HREF}/manifests/02-dedicated-admin-operator-cluster.ClusterRoleBinding.yaml
oc apply -f ${BASE_HREF}/manifests/02-dedicated-admin-operator-project.ClusterRoleBinding.yaml
oc apply -f ${BASE_HREF}/manifests/02-dedicated-admins-cluster.ClusterRoleBinding.yaml
oc apply -f ${BASE_HREF}/manifests/10-dedicated-admin-operator.Deployment.yaml
oc apply -f ${BASE_HREF}/manifests/10-dedicated-admin-operator.Role.yaml
oc apply -f ${BASE_HREF}/manifests/11-dedicated-admin-operator.RoleBinding.yaml