#OCM_IMAGE=registry.svc.ci.openshift.org/openshift/release:intly-golang-1.12
#OCM=docker run --rm -it -u 1000 -v "/home/mnairn/go/src/github.com/integr8ly/integreatly-operator:/integreatly-operator/" -w "/integreatly-operator" -v "${HOME}/tmp-home:/myhome:z" -e "HOME=/myhome" --entrypoint=/usr/local/bin/ocm ${OCM_IMAGE}
OCM=ocm
OCM_CLUSTER_NAME=rhmi-$(date +"%y%m%d_%H%M")

.PHONY: ocm/version
ocm/version:
	@${OCM} version

ocm/login: export OCM_URL := https://api.stage.openshift.com/
ocm/login: export OCM_TOKEN := ""
.PHONY: ocm/login
ocm/login:
	@${OCM} login --url=$(OCM_URL) --token=$(OCM_TOKEN)

.PHONY: ocm/whoami
ocm/whoami:
	@${OCM} whoami

.PHONY: ocm/execute
ocm/execute:
	${OCM} ${CMD}

.PHONY: ocm/get/current_account
ocm/get/current_account:
	@${OCM} get /api/accounts_mgmt/v1/current_account

.PHONY: ocm/cluster/list
ocm/cluster/list:
	@${OCM} cluster list

.PHONY: ocm/cluster/create
ocm/cluster/create:
	@$(eval OCM_CLUSTER_ID=$(shell ${OCM} post /api/clusters_mgmt/v1/clusters --body=ocm/cluster.json | jq -r .id ))
	@echo "Cluster creation started. Cluster id: ${OCM_CLUSTER_ID}"
	$(call wait_command, $(OCM) get /api/clusters_mgmt/v1/clusters/${OCM_CLUSTER_ID}/status | jq -r .state | grep -q ready, cluster creation, 120m, 300)

.PHONY: ocm/cluster/delete
ocm/cluster/delete:
	${OCM} delete /api/clusters_mgmt/v1/clusters/$(OCM_CLUSTER_ID)

.PHONY: ocm/cluster.json
ocm/cluster.json: export OCM_CLUSTER_NAME := ""
ocm/cluster.json: export OCM_CLUSTER_REGION := "eu-west-1"
ocm/cluster.json:
	@mkdir -p ocm
	sed "s/OCM_CLUSTER_NAME/$(OCM_CLUSTER_NAME)/g" templates/ocm-cluster/cluster-template.json | \
	sed "s/OCM_CLUSTER_REGION/$(OCM_CLUSTER_REGION)/g" > ocm/cluster.json