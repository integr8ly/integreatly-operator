include ./make/*.mk

ORG ?= integreatly
NAMESPACE=redhat-rhmi-operator
PROJECT=integreatly-operator
REG=quay.io
SHELL=/bin/bash
TAG ?= 2.2.0
PKG=github.com/integr8ly/integreatly-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=integreatly-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
OPERATOR_SDK_VERSION=0.15.1
AUTH_TOKEN=$(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "$(QUAY_USERNAME)", "password": "${QUAY_PASSWORD}"}}' | jq -r '.token')
TEMPLATE_PATH="$(shell pwd)/templates/monitoring"
INTEGREATLY_OPERATOR_IMAGE ?= $(REG)/$(ORG)/$(PROJECT):v$(TAG)
CONTAINER_ENGINE ?= docker

# If openapi-gen is available on the path, use that; otherwise use it through
# "go run" (slower)
ifneq (, $(shell which openapi-gen 2> /dev/null))
	OPENAPI_GEN ?= openapi-gen
else
	OPENAPI_GEN ?= go run k8s.io/kube-openapi/cmd/openapi-gen
endif

# If the _correct_ version of operator-sdk is on the path, use that (faster);
# otherwise use it through "go run" (slower but will always work and will use correct version)
ifeq ($(shell operator-sdk version 2> /dev/null | sed -e 's/", .*/"/' -e 's/.* //'), "v$(OPERATOR_SDK_VERSION)")
	OPERATOR_SDK ?= operator-sdk
else
	OPERATOR_SDK ?= go run github.com/operator-framework/operator-sdk/cmd/operator-sdk
endif

# Set sed -i as it's different for mac vs gnu
ifeq ($(shell uname -s | tr A-Z a-z), darwin)
	SED_INLINE ?= sed -i ''
else
 	SED_INLINE ?= sed -i
endif

export SELF_SIGNED_CERTS   ?= true
export INSTALLATION_TYPE   ?= managed
export INSTALLATION_NAME   ?= rhmi
export INSTALLATION_PREFIX ?= redhat-rhmi
export USE_CLUSTER_STORAGE ?= true
export OPERATORS_IN_PRODUCT_NAMESPACE ?= false # e2e tests and createInstallationCR() need to be updated when default is changed
export DELOREAN_PULL_SECRET_NAME ?= integreatly-delorean-readonly-pull-secret

define wait_command
	@echo Waiting for $(2) for $(3)...
	@time timeout --foreground $(3) bash -c "until $(1); do echo $(2) not ready yet, trying again in $(4)s...; sleep $(4); done"
	@echo $(2) ready!
endef

.PHONY: setup/moq
setup/moq:
	go install github.com/matryer/moq

.PHONY: setup/service_account
setup/service_account:
	@-oc new-project $(NAMESPACE)
	@oc project $(NAMESPACE)
	@oc replace --force -f deploy/role.yaml
	@oc replace --force -f deploy/service_account.yaml -n $(NAMESPACE)
	@cat deploy/role_binding.yaml | sed "s/namespace: integreatly/namespace: $(NAMESPACE)/g" | oc replace --force -f -

.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

.PHONY: code/run
code/run: code/gen cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty
	@$(OPERATOR_SDK) run --local --namespace="$(NAMESPACE)"

.PHONY: code/rerun
code/rerun:
	@$(OPERATOR_SDK) run --local --namespace="$(NAMESPACE)"

.PHONY: code/run/service_account
code/run/service_account: setup/service_account
	@oc login --token=$(shell oc serviceaccounts get-token rhmi-operator -n ${NAMESPACE})
	$(MAKE) code/run

.PHONY: code/run/delorean
code/run/delorean: cluster/cleanup cluster/prepare cluster/prepare/local deploy/integreatly-rhmi-cr.yml code/run/service_account

.PHONY: code/compile
code/compile: code/gen
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/manager

deploy/crds/integreatly.org_rhmis_crd.yaml: pkg/apis/integreatly/v1alpha1/rhmi_types.go
	$(OPERATOR_SDK) generate crds

pkg/apis/integreatly/v1alpha1/zz_generated.openapi.go: pkg/apis/integreatly/v1alpha1/rhmi_types.go
	$(OPENAPI_GEN) --logtostderr=true -o "" \
		-i ./pkg/apis/integreatly/v1alpha1/ \
		-p ./pkg/apis/integreatly/v1alpha1/ \
		-O zz_generated.openapi \
		-h ./hack/boilerplate.go.txt \
		-r "-"

pkg/apis/integreatly/v1alpha1/zz_generated.deepcopy.go:	pkg/apis/integreatly/v1alpha1/rhmi_types.go
	$(OPERATOR_SDK) generate k8s

.PHONY: code/gen
code/gen: setup/moq deploy/crds/integreatly.org_rhmis_crd.yaml pkg/apis/integreatly/v1alpha1/zz_generated.deepcopy.go pkg/apis/integreatly/v1alpha1/zz_generated.openapi.go
	find ./ -name *_moq.go -type f -not -path "./vendor/*" -delete
	@go generate ./...

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)
	golint ./pkg/... | grep -v  "comment on" | grep -v "or be unexported"
	go vet ./...

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/compile
	@$(OPERATOR_SDK) build $(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: image/push
image/push:
	docker push $(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: image/build/test
image/build/test:
	$(OPERATOR_SDK) build --enable-tests $(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: test/unit
test/unit:
	@TEMPLATE_PATH=$(TEMPLATE_PATH) ./scripts/ci/unit_test.sh

.PHONY: test/e2e/prow
test/e2e/prow: export SURF_DEBUG_HEADERS=1
test/e2e/prow: export component := integreatly-operator
test/e2e/prow: export INTEGREATLY_OPERATOR_IMAGE := "${IMAGE_FORMAT}"
test/e2e/prow: test/e2e

.PHONY: test/e2e
test/e2e:  export SURF_DEBUG_HEADERS=1
test/e2e:  cluster/cleanup cluster/cleanup/crds cluster/prepare cluster/prepare/crd cluster/prepare/service deploy/integreatly-rhmi-cr.yml
	 export SURF_DEBUG_HEADERS=1
	$(OPERATOR_SDK) --verbose test local ./test/e2e --namespace="$(NAMESPACE)" --go-test-flags "-timeout=60m" --debug --image=$(INTEGREATLY_OPERATOR_IMAGE)

.PHONY: test/e2e/local
test/e2e/local: cluster/cleanup cluster/cleanup/crds cluster/prepare cluster/prepare/crd deploy/integreatly-rhmi-cr.yml
	$(OPERATOR_SDK) --verbose test local ./test/e2e --namespace="$(NAMESPACE)" --go-test-flags "-timeout=60m" --debug --up-local


.PHONY: test/functional
test/functional:  export SURF_DEBUG_HEADERS=1
test/functional:
	# Run the functional tests against an existing cluster. Make sure you have logged in to the cluster.
	go clean -testcache && go test -v ./test/functional -timeout=80m

.PHONY: install/olm
install/olm: cluster/cleanup/olm cluster/cleanup/crds cluster/prepare cluster/prepare/olm/subscription deploy/integreatly-rhmi-cr.yml cluster/check/operator/deployment cluster/prepare/dms cluster/prepare/pagerduty

.PHONY: test/e2e/olm
test/e2e/olm: install/olm
#ToDo Trigger test suite here

.PHONY: cluster/deploy/integreatly-rhmi-cr.yml
cluster/deploy/integreatly-rhmi-cr.yml: deploy/integreatly-rhmi-cr.yml
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.bootstrap.phase}' | grep -q completed, bootstrap phase, 5m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.monitoring.phase}' | grep -q completed, monitoring phase, 10m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.authentication.phase}' | grep -q completed, authentication phase, 10m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.products.phase}' | grep -q completed, products phase, 30m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.solution-explorer.phase}' | grep -q completed, solution-explorer phase, 10m, 30)

.PHONY: cluster/prepare
cluster/prepare: cluster/prepare/project cluster/prepare/osrc cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty delorean/cluster/prepare

.PHONY: cluster/prepare/project
cluster/prepare/project:
	@ - oc new-project $(NAMESPACE)
	@oc label namespace $(NAMESPACE) monitoring-key=middleware --overwrite
	@oc project $(NAMESPACE)

.PHONY: cluster/prepare/configmaps
cluster/prepare/configmaps:
	@oc process -f deploy/cro-configmaps.yaml -p INSTALLATION_NAMESPACE=$(NAMESPACE) | oc apply -f -

.PHONY: cluster/prepare/osrc
cluster/prepare/osrc:
	- oc process -p NAMESPACE=$(NAMESPACE) OPERATOR_SOURCE_REGISTRY_NAMESPACE=$(ORG) -f deploy/operator-source-template.yml | oc apply -f - -n openshift-marketplace

.PHONY: cluster/prepare/crd
cluster/prepare/crd:
	- oc create -f deploy/crds/integreatly.org_rhmis_crd.yaml
	- oc create -f deploy/crds/integreatly.org_rhmiconfigs_crd.yaml

.PHONY: cluster/prepare/service
cluster/prepare/service:
	- oc create -f deploy/webhook-service.yaml

.PHONY: cluster/prepare/local
cluster/prepare/local: cluster/prepare/project cluster/prepare/crd cluster/prepare/service cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/delorean
	@oc create -f deploy/service_account.yaml
	@oc create -f deploy/role.yaml
	@oc create -f deploy/role_binding.yaml

.PHONY: cluster/prepare/olm/subscription
cluster/prepare/olm/subscription:
	oc process -p NAMESPACE=$(NAMESPACE) -f deploy/operator-subscription-template.yml | oc create -f - -n $(NAMESPACE)
	$(call wait_command, oc get crd rhmis.integreatly.org, rhmis.integreatly.org crd, 1m, 10)

.PHONY: cluster/check/operator/deployment
cluster/check/operator/deployment:
	$(call wait_command, oc get deployments rhmi-operator -n $(NAMESPACE) --output=json -o jsonpath='{.status.availableReplicas}' | grep -q 1, rhmi-operator ,2m, 10)

.PHONY: cluster/prepare/olm
cluster/prepare/olm: cluster/prepare cluster/prepare/olm/subscription cluster/check/operator/deployment

.PHONY: cluster/prepare/smtp
cluster/prepare/smtp:
	@-oc create secret generic $(INSTALLATION_PREFIX)-smtp -n $(NAMESPACE) \
		--from-literal=host=smtp.example.com \
		--from-literal=username=dummy \
		--from-literal=password=dummy \
		--from-literal=port=587 \
		--from-literal=tls=true

.PHONY: cluster/prepare/pagerduty
cluster/prepare/pagerduty:
	@-oc create secret generic $(INSTALLATION_PREFIX)-pagerduty -n $(NAMESPACE) \
		--from-literal=serviceKey=test

.PHONY: cluster/prepare/dms
cluster/prepare/dms:
	@-oc create secret generic $(INSTALLATION_PREFIX)-deadmanssnitch -n $(NAMESPACE) \
		--from-literal=url=https://dms.example.com

.PHONY: cluster/prepare/delorean
cluster/prepare/delorean:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
	@echo Detected a delorean ews branch. The integreatly-delorean-secret.yml is required.
	@echo Please contact the delorean team to get this if you do not already have it.
	@echo Add it to the root dir of this repo and rerun the desired target if the target fails on it not existing
	@ oc apply -f integreatly-delorean-secret.yml --namespace=$(NAMESPACE)
endif

.PHONY: cluster/prepare/delorean/pullsecret
cluster/prepare/delorean/pullsecret:
	@./scripts/setup-delorean-pullsecret.sh

.PHONY: cluster/cleanup
cluster/cleanup:
	@-oc delete -f deploy/integreatly-rhmi-cr.yml --timeout=240s --wait
	@-oc delete namespace $(NAMESPACE) --timeout=60s --wait
	@-oc delete -f deploy/role.yaml
	@-oc delete -f deploy/role_binding.yaml

.PHONY: cluster/cleanup/olm
cluster/cleanup/olm: cluster/cleanup
	$(call wait_command, oc get projects -l integreatly=true -o jsonpath='{.items}' | grep -q '\[\]', integreatly namespace cleanup, 4m, 10)
	@-oc delete catalogsourceconfig.operators.coreos.com/installed-rhmi-operator -n openshift-marketplace
	@-oc delete operatorsource.operators.coreos.com/rhmi-operators -n openshift-marketplace

.PHONY: cluster/cleanup/crds
cluster/cleanup/crds:
	@-oc delete crd applicationmonitorings.applicationmonitoring.integreatly.org
	@-oc delete crd blackboxtargets.applicationmonitoring.integreatly.org
	@-oc delete crd grafanadashboards.integreatly.org
	@-oc delete crd grafanadatasources.integreatly.org
	@-oc delete crd grafanas.integreatly.org
	@-oc delete crd rhmis.integreatly.org
	@-oc delete crd webapps.integreatly.org
	@-oc delete crd rhmiconfigs.integreatly.org

.PHONY: deploy/integreatly-rhmi-cr.yml
deploy/integreatly-rhmi-cr.yml:
	@echo "selfSignedCerts = $(SELF_SIGNED_CERTS)"
	sed "s/INSTALLATION_NAME/$(INSTALLATION_NAME)/g" deploy/crds/examples/integreatly-rhmi-cr.yaml | \
	sed "s/INSTALLATION_TYPE/$(INSTALLATION_TYPE)/g" | \
	sed "s/INSTALLATION_PREFIX/$(INSTALLATION_PREFIX)/g" | \
	sed "s/SELF_SIGNED_CERTS/$(SELF_SIGNED_CERTS)/g" | \
	sed "s/OPERATORS_IN_PRODUCT_NAMESPACE/$(OPERATORS_IN_PRODUCT_NAMESPACE)/g" | \
	sed "s/USE_CLUSTER_STORAGE/$(USE_CLUSTER_STORAGE)/g" > deploy/integreatly-rhmi-cr.yml
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
	@sed -i.bak "s/DELOREAN_PULL_SECRET_NAMESPACE/$(NAMESPACE)/g" deploy/integreatly-rhmi-cr.yml
	@sed -i.bak "s/DELOREAN_PULL_SECRET_NAME/$(DELOREAN_PULL_SECRET_NAME)/g" deploy/integreatly-rhmi-cr.yml
	rm deploy/integreatly-rhmi-cr.yml.bak
else
	@sed -i.bak "s/DELOREAN_PULL_SECRET_NAMESPACE//g" deploy/integreatly-rhmi-cr.yml
	@sed -i.bak "s/DELOREAN_PULL_SECRET_NAME//g" deploy/integreatly-rhmi-cr.yml
	rm deploy/integreatly-rhmi-cr.yml.bak
endif
	@-oc create -f deploy/integreatly-rhmi-cr.yml

.PHONY: prepare-patch-release
prepare-patch-release:
	$(CONTAINER_ENGINE) pull quay.io/integreatly/delorean-cli:master
	$(CONTAINER_ENGINE) run --rm -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z -v "${HOME}/.delorean.yaml:/.delorean.yaml" quay.io/integreatly/delorean-cli:master delorean release openshift-ci-release --config /.delorean.yaml --version $(TAG)

.PHONY: release/prepare
release/prepare:
	@./scripts/prepare-release.sh

.PHONY: push/csv
push/csv:
	operator-courier verify deploy/olm-catalog/integreatly-operator
	-operator-courier push deploy/olm-catalog/integreatly-operator/ $(REPO) integreatly $(TAG) "$(AUTH_TOKEN)"

.PHONY: gen/push/csv
gen/push/csv: release/prepare push/csv

# Generate namespace names to be used in docs
.PHONY: gen/namespaces
gen/namespaces:
	echo '// Generated file. Do not edit' > namespaces.asciidoc
	oc get namespace | \
	grep redhat-rhmi | \
	awk -S '{print"- "$$1}' >> namespaces.asciidoc

.PHONY: vendor/check
vendor/check: vendor/fix
	git diff --exit-code vendor/

.PHONY: vendor/fix
vendor/fix:
	go mod tidy
	go mod vendor