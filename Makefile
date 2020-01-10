ORG=integreatly
NAMESPACE=integreatly
PROJECT=integreatly-operator
REG=quay.io
SHELL=/bin/bash
PREVIOUS_TAG=1.14.0
TAG=1.15.0
PKG=github.com/integr8ly/integreatly-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=integreatly-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
OPERATOR_SDK_VERSION=0.12.0
AUTH_TOKEN=$(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "$(QUAY_USERNAME)", "password": "${QUAY_PASSWORD}"}}' | jq -r '.token')
TEMPLATE_PATH="$(shell pwd)/templates/monitoring"
INTEGREATLY_OPERATOR_IMAGE ?= $(REG)/$(ORG)/$(PROJECT):v$(TAG)

define wait_command
	@echo Waiting for $(2) for $(3)...
	@time timeout --foreground $(3) bash -c "until $(1); do echo $(2) not ready yet, trying again in $(4)...; sleep $(4); done"
	@echo $(2) ready!
endef

.PHONY: setup/moq
setup/moq:
	go get github.com/matryer/moq

.PHONY: setup/travis
setup/travis:
	@echo Installing Operator SDK
	@curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v$(OPERATOR_SDK_VERSION)/operator-sdk-v$(OPERATOR_SDK_VERSION)-x86_64-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/

.PHONY: setup/service_account
setup/service_account:
	@oc replace --force -f deploy/role.yaml -n $(NAMESPACE)
	@oc replace --force -f deploy/service_account.yaml -n $(NAMESPACE)
	@oc replace --force -f deploy/role_binding.yaml -n $(NAMESPACE)
	@cat deploy/role_binding.yaml | sed "s/namespace: integreatly/namespace: $(NAMESPACE)/g" | oc replace --force -f -

.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

.PHONY: code/run
code/run:
	@export CR_NAMESPACE=${NAMESPACE}; operator-sdk up local --namespace=""

.PHONY: code/run/service_account
code/run/service_account: setup/service_account
	@oc login --token=$(shell oc serviceaccounts get-token integreatly-operator -n ${NAMESPACE})
	$(MAKE) code/run

.PHONY: code/compile
code/compile:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/manager

.PHONY: code/gen
code/gen:
	find ./ -name *_moq.go -type f -delete
	operator-sdk generate k8s
	operator-sdk generate openapi
	@go generate ./...

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)
	go vet ./...

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/compile
	@operator-sdk build $(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: image/push
image/push:
	docker push $(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: image/build/test
image/build/test:
	operator-sdk build --enable-tests $(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: test/unit
test/unit:
	TEMPLATE_PATH=$(TEMPLATE_PATH) ./scripts/ci/unit_test.sh

.PHONY: test/e2e/prow
test/e2e/prow: export INTEGREATLY_OPERATOR_IMAGE := "registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:integreatly-operator"
test/e2e/prow: test/e2e

.PHONY: test/e2e
test/e2e: export GH_CLIENT_ID := 1234
test/e2e: export GH_CLIENT_SECRET := 1234
test/e2e: cluster/cleanup cluster/cleanup/crds cluster/prepare cluster/prepare/configmaps cluster/prepare/crd deploy/integreatly-installation-cr.yml
	operator-sdk --verbose test local ./test/e2e --namespace="$(NAMESPACE)" --go-test-flags "-timeout=60m" --debug --image=$(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: test/e2e/olm
test/e2e/olm: export GH_CLIENT_ID := 1234
test/e2e/olm: export GH_CLIENT_SECRET := 1234
test/e2e/olm: cluster/cleanup/olm cluster/prepare/olm cluster/prepare/configmaps cluster/deploy/integreatly-installation-cr.yml

.PHONY: cluster/deploy/integreatly-installation-cr.yml
cluster/deploy/integreatly-installation-cr.yml: export INSTALLATION_NAME := integreatly-operator
cluster/deploy/integreatly-installation-cr.yml: deploy/integreatly-installation-cr.yml
	$(call wait_command, oc get Installation $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.bootstrap.phase}' | grep -q completed, bootstrap phase, 5m, 30)
	$(call wait_command, oc get Installation $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.monitoring.phase}' | grep -q completed, monitoring phase, 10m, 30)
	$(call wait_command, oc get Installation $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.authentication.phase}' | grep -q completed, authentication phase, 10m, 30)
	$(call wait_command, oc get Installation $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.products.phase}' | grep -q completed, products phase, 30m, 30)
	$(call wait_command, oc get Installation $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.solution-explorer.phase}' | grep -q completed, solution-explorer phase, 10m, 30)

.PHONY: cluster/prepare
cluster/prepare: cluster/prepare/project cluster/prepare/secrets cluster/prepare/osrc

.PHONY: cluster/prepare/project
cluster/prepare/project:
	@oc new-project $(NAMESPACE)
	@oc label namespace $(NAMESPACE) monitoring-key=middleware
	@oc project $(NAMESPACE)

.PHONY: cluster/prepare/secrets
cluster/prepare/secrets:
	@oc create secret generic github-oauth-secret \
		--from-literal=clientId=$(GH_CLIENT_ID) \
		--from-literal=secret=$(GH_CLIENT_SECRET)

.PHONY: cluster/prepare/configmaps
cluster/prepare/configmaps:
	@oc process -f deploy/cro-configmaps.yaml -p INSTALLATION_NAMESPACE=$(NAMESPACE) | oc apply -f -

.PHONY: cluster/prepare/osrc
cluster/prepare/osrc:
	- oc process -p NAMESPACE=$(NAMESPACE) OPERATOR_SOURCE_REGISTRY_NAMESPACE=$(ORG) -f deploy/operator-source-template.yml | oc apply -f - -n openshift-marketplace

.PHONY: cluster/prepare/crd
cluster/prepare/crd:
	- oc create -f deploy/crds/*_crd.yaml

.PHONY: cluster/prepare/local
cluster/prepare/local: cluster/prepare/project cluster/prepare/secrets cluster/prepare/crd
	@oc create -f deploy/service_account.yaml
	@oc create -f deploy/role.yaml
	@oc create -f deploy/role_binding.yaml

.PHONY: cluster/prepare/olm
cluster/prepare/olm: cluster/prepare/project cluster/prepare/secrets cluster/prepare/osrc
	oc process -p NAMESPACE=$(NAMESPACE) -f deploy/operator-subscription-template.yml | oc create -f - -n $(NAMESPACE)
	$(call wait_command, oc get crd installations.integreatly.org, installations.integreatly.org crd, 1m, 10)
	$(call wait_command, oc get deployments integreatly-operator -n $(NAMESPACE) --output=json -o jsonpath='{.status.availableReplicas}' | grep -q 1, integreatly-operator ,2m, 10)

.PHONY: cluster/cleanup
cluster/cleanup:
	@-oc delete namespace $(NAMESPACE) --timeout=240s --wait
	@-oc delete catalogsourceconfig.operators.coreos.com/installed-integreatly-operator -n openshift-marketplace
	@-oc delete operatorsource.operators.coreos.com/integreatly-operators -n openshift-marketplace

.PHONY: cluster/cleanup/olm
cluster/cleanup/olm:
	@-oc delete -f deploy/integreatly-installation-cr.yml --timeout=60s --wait
	@-oc delete namespace $(NAMESPACE) --timeout=240s --wait
	$(call wait_command, oc get projects -l integreatly=true -o jsonpath='{.items}' | grep -q '\[\]', integreatly namespace cleanup, 4m, 10)
	@-oc delete catalogsourceconfig.operators.coreos.com/installed-integreatly-operator -n openshift-marketplace
	@-oc delete operatorsource.operators.coreos.com/integreatly-operators -n openshift-marketplace

.PHONY: cluster/cleanup/crds
cluster/cleanup/crds:
	@-oc delete crd applicationmonitorings.applicationmonitoring.integreatly.org
	@-oc delete crd blackboxtargets.applicationmonitoring.integreatly.org
	@-oc delete crd grafanadashboards.integreatly.org
	@-oc delete crd grafanadatasources.integreatly.org
	@-oc delete crd grafanas.integreatly.org
	@-oc delete crd installations.integreatly.org
	@-oc delete crd webapps.integreatly.org

.PHONY: deploy/integreatly-installation-cr.yml
deploy/integreatly-installation-cr.yml: export SELF_SIGNED_CERTS := true
deploy/integreatly-installation-cr.yml: export INSTALLATION_NAME := integreatly-operator
deploy/integreatly-installation-cr.yml: export INSTALLATION_TYPE := managed
deploy/integreatly-installation-cr.yml: export INSTALLATION_PREFIX := rhmi
deploy/integreatly-installation-cr.yml:
	@echo "selfSignedCerts = $(SELF_SIGNED_CERTS)"
	sed "s/INSTALLATION_NAME/$(INSTALLATION_NAME)/g" deploy/crds/examples/integreatly-installation-cr.yaml | \
	sed "s/INSTALLATION_TYPE/$(INSTALLATION_TYPE)/g" | \
	sed "s/INSTALLATION_PREFIX/$(INSTALLATION_PREFIX)/g" | \
	sed "s/SELF_SIGNED_CERTS/$(SELF_SIGNED_CERTS)/g" > deploy/integreatly-installation-cr.yml
	@-oc create -f deploy/integreatly-installation-cr.yml

.PHONY: gen/csv
gen/csv:
	@mv deploy/olm-catalog/integreatly-operator/integreatly-operator-$(PREVIOUS_TAG) deploy/olm-catalog/integreatly-operator/$(PREVIOUS_TAG)
	@rm -rf deploy/olm-catalog/integreatly-operator/integreatly-operator-$(TAG)
	@sed -i 's/image:.*/image: quay\.io\/integreatly\/integreatly-operator:v$(TAG)/g' deploy/operator.yaml
	operator-sdk olm-catalog gen-csv --csv-version $(TAG) --default-channel --csv-channel=integreatly --update-crds --from-version $(PREVIOUS_TAG)
	@echo Updating package file
	@sed -i 's/$(PREVIOUS_TAG)/$(TAG)/g' version/version.go
	@sed -i 's/$(PREVIOUS_TAG)/$(TAG)/g' deploy/olm-catalog/integreatly-operator/integreatly-operator.package.yaml
	@mv deploy/olm-catalog/integreatly-operator/$(PREVIOUS_TAG) deploy/olm-catalog/integreatly-operator/integreatly-operator-$(PREVIOUS_TAG)
	@mv deploy/olm-catalog/integreatly-operator/$(TAG) deploy/olm-catalog/integreatly-operator/integreatly-operator-$(TAG)
	@sed -i 's/integreatly-operator:v$(PREVIOUS_TAG)/integreatly-operator:v$(TAG)/g' deploy/olm-catalog/integreatly-operator/integreatly-operator-$(TAG)/integreatly-operator.v${TAG}.clusterserviceversion.yaml

.PHONY: push/csv
push/csv:
	operator-courier verify deploy/olm-catalog/integreatly-operator
	-operator-courier push deploy/olm-catalog/integreatly-operator/ $(REPO) integreatly $(TAG) "$(AUTH_TOKEN)"

.PHONY: gen/push/csv
gen/push/csv: gen/csv push/csv

.PHONY: vendor/check
vendor/check: vendor/fix
	git diff --exit-code vendor/

.PHONY: vendor/fix
vendor/fix:
	go mod tidy
	go mod vendor
