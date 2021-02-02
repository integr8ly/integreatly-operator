include ./make/*.mk

ORG ?= integreatly

REG=quay.io
SHELL=/bin/bash

PKG=github.com/integr8ly/integreatly-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=integreatly-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
OPERATOR_SDK_VERSION=1.2.0
AUTH_TOKEN=$(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "$(QUAY_USERNAME)", "password": "$(QUAY_PASSWORD)"}}' | jq -r '.token')
TEMPLATE_PATH="$(shell pwd)/templates/monitoring"
IN_PROW ?= "false"
TYPE_OF_MANIFEST ?= master

CONTAINER_ENGINE ?= docker
TEST_RESULTS_DIR ?= test-results
TEMP_SERVICEACCOUNT_NAME=rhmi-operator

# These tags are modified by the prepare-release script.
RHMI_TAG ?= 2.8.0
RHOAM_TAG ?= 1.2.0

export SKIP_FLAKES := true

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
# Setting the INSTALLATION_TYPE to managed-api will configure the values required for RHOAM installs
export INSTALLATION_TYPE   ?= managed

export ALERT_SMTP_FROM ?= noreply-alert@devshift.org
export USE_CLUSTER_STORAGE ?= true
export OPERATORS_IN_PRODUCT_NAMESPACE ?= false # e2e tests and createInstallationCR() need to be updated when default is changed
export DELOREAN_PULL_SECRET_NAME ?= integreatly-delorean-pull-secret
export ALERTING_EMAIL_ADDRESS ?= noreply-test@rhmi-redhat.com
export BU_ALERTING_EMAIL_ADDRESS ?= noreply-test@rhmi-redhat.com

ifeq ($(INSTALLATION_TYPE), managed)
	PROJECT=integreatly-operator
	TAG ?= $(RHMI_TAG)
	OPERATOR_IMAGE=$(REG)/$(ORG)/$(PROJECT):v$(TAG)
	NAMESPACE_PREFIX ?= redhat-rhmi-
	APPLICATION_REPO ?= integreatly
	export INSTALLATION_PREFIX ?= redhat-rhmi
	export OLM_TYPE ?= integreatly-operator
	INSTALLATION_NAME ?= rhmi
	INSTALLATION_SHORTHAND ?= rhmi
endif

ifeq ($(INSTALLATION_TYPE), managed-api)
	PROJECT=managed-api-service
	TAG ?= $(RHOAM_TAG)
	OPERATOR_IMAGE=$(REG)/$(ORG)/$(PROJECT):v$(TAG)
	NAMESPACE_PREFIX ?= redhat-rhoam-
	APPLICATION_REPO ?= managed-api-service
	# TODO follow on naming of this folder by INSTALLATION_PREFIX and contents of the role_binding.yaml
	export INSTALLATION_PREFIX ?= redhat-rhoam
	export OLM_TYPE ?= managed-api-service
	INSTALLATION_NAME ?= rhoam
	INSTALLATION_SHORTHAND ?= rhoam
endif

NAMESPACE=$(NAMESPACE_PREFIX)operator

define wait_command
	@echo Waiting for $(2) for $(3)...
	@time timeout --foreground $(3) bash -c "until $(1); do echo $(2) not ready yet, trying again in $(4)s...; sleep $(4); done"
	@echo $(2) ready!
endef

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: setup/moq
setup/moq:
	go install github.com/matryer/moq

.PHONY: setup/service_account
setup/service_account: kustomize
	@-oc new-project $(NAMESPACE)
	@oc project $(NAMESPACE)
	@-oc create -f config/rbac/service_account.yaml -n $(NAMESPACE)
	@$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc replace --force -f -	
	@oc login --token=$(shell oc serviceaccounts get-token rhmi-operator -n ${NAMESPACE}) --server=$(shell sh -c "oc cluster-info | grep -Eo 'https?://[-a-zA-Z0-9\.:]*'") --kubeconfig=TMP_SA_KUBECONFIG --insecure-skip-tls-verify=true

.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

.PHONY: code/run
code/run: code/gen cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty setup/service_account
	@KUBECONFIG=TMP_SA_KUBECONFIG WATCH_NAMESPACE=$(NAMESPACE) go run ./main.go

.PHONY: code/rerun
code/rerun: setup/service_account
	@KUBECONFIG=TMP_SA_KUBECONFIG WATCH_NAMESPACE=$(NAMESPACE) go run ./main.go

.PHONY: code/run/service_account
code/run/service_account: code/run

.PHONY: code/run/delorean
code/run/delorean: cluster/cleanup cluster/prepare cluster/prepare/local deploy/integreatly-rhmi-cr.yml code/run/service_account

.PHONY: code/compile
code/compile: code/gen
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) .

deploy/crds/integreatly.org_rhmis_crd.yaml: controller-gen apis/v1alpha1/rhmi_types.go
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

pkg/apis/integreatly/v1alpha1/zz_generated.openapi.go: apis/v1alpha1/rhmi_types.go
	$(OPENAPI_GEN) --logtostderr=true -o "" \
		-i ./apis/v1alpha1/ \
		-p ./apis/v1alpha1/ \
		-O zz_generated.openapi \
		-h ./hack/boilerplate.go.txt \
		-r "-"

apis/integreatly/v1alpha1/zz_generated.deepcopy.go: controller-gen apis/v1alpha1/rhmi_types.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: code/gen
code/gen: setup/moq deploy/crds/integreatly.org_rhmis_crd.yaml apis/integreatly/v1alpha1/zz_generated.deepcopy.go
	$(CONTROLLER_GEN) rbac:roleName=manager-role webhook paths="./..."
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
	echo "build image $(OPERATOR_IMAGE)"
	docker build . -t ${OPERATOR_IMAGE}

.PHONY: image/push
image/push:
	echo "push image $(OPERATOR_IMAGE)"
	docker push $(OPERATOR_IMAGE)

.PHONY: image/build/push
image/build/push: image/build image/push


############ E2E TEST COMMANDS ############
.PHONY: test/e2e/prow
test/e2e/prow: export component := integreatly-operator
test/e2e/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/prow: export INSTALLATION_TYPE := managed
test/e2e/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/prow: export WATCH_NAMESPACE := redhat-rhmi-operator
test/e2e/prow: export NAMESPACE_PREFIX := redhat-rhmi-
test/e2e/prow: export INSTALLATION_PREFIX := redhat-rhmi
test/e2e/prow: export INSTALLATION_NAME := rhmi
test/e2e/prow: export INSTALLATION_SHORTHAND := rhmi
test/e2e/prow: IN_PROW = "true"
test/e2e/prow: test/e2e

.PHONY: test/e2e/rhoam/prow
test/e2e/rhoam/prow: export component := integreatly-operator
test/e2e/rhoam/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/rhoam/prow: export INSTALLATION_TYPE := managed-api
test/e2e/rhoam/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/rhoam/prow: export WATCH_NAMESPACE := redhat-rhoam-operator
test/e2e/rhoam/prow: export NAMESPACE_PREFIX := redhat-rhoam-
test/e2e/rhoam/prow: export INSTALLATION_PREFIX := redhat-rhoam
test/e2e/rhoam/prow: export INSTALLATION_NAME := rhoam
test/e2e/rhoam/prow: export INSTALLATION_SHORTHAND := rhoam
test/e2e/rhoam/prow: IN_PROW = "true"
test/e2e/rhoam/prow: test/e2e

.PHONY: test/e2e
test/e2e: export SURF_DEBUG_HEADERS=1
test/e2e: cluster/deploy
	go clean -testcache && go test -v ./test/e2e -timeout=120m -ginkgo.v

.PHONY: test/e2e/single
test/e2e/single: export WATCH_NAMESPACE := $(NAMESPACE)
test/e2e/single: 
	go clean -testcache && go test -v ./test/functional -run="//^$(TEST)" -timeout=80m

.PHONY: test/functional
test/functional: export WATCH_NAMESPACE := $(NAMESPACE)
test/functional:
	# Run the functional tests against an existing cluster. Make sure you have logged in to the cluster.
	go clean -testcache && go test -v ./test/functional -timeout=80m

.PHONY: test/osde2e
test/osde2e: export WATCH_NAMESPACE := $(NAMESPACE)
test/osde2e: export SKIP_FLAKES := $(SKIP_FLAKES)
test/osde2e:
	# Run the osde2e tests against an existing cluster. Make sure you have logged in to the cluster.
	go clean -testcache && go test -v ./test/osde2e -timeout=120m

############ E2E TEST COMMANDS ############


.PHONY: test/products/local
test/products/local: export WATCH_NAMESPACE := $(NAMESPACE)
test/products/local:
	# Running the products tests against an existing cluster inside a container. Make sure you have logged in to the cluster.
	# Using 'test-containers.yaml' as config and 'test-results' as output dir
	mkdir -p "test-results"
	$(CONTAINER_ENGINE) pull quay.io/integreatly/delorean-cli:master
	$(CONTAINER_ENGINE) run --rm -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z -v $(shell pwd)/test-containers.yaml:/test-containers.yaml -v $(shell pwd)/test-results:/test-results quay.io/integreatly/delorean-cli:master delorean pipeline product-tests --test-config ./test-containers.yaml --output /test-results --namespace test-products

.PHONY: test/products
test/products: export WATCH_NAMESPACE := $(NAMESPACE)
test/products:
	# Running the products tests against an existing cluster. Make sure you have logged in to the cluster.
	# Using "test-containers.yaml" as config and $(TEST_RESULTS_DIR) as output dir
	mkdir -p $(TEST_RESULTS_DIR)
	delorean pipeline product-tests --test-config ./test-containers.yaml --output $(TEST_RESULTS_DIR) --namespace test-products

.PHONY: test/rhoam/products
test/rhoam/products: export WATCH_NAMESPACE := $(NAMESPACE)
test/rhoam/products:
	mkdir -p $(TEST_RESULTS_DIR)
	delorean pipeline product-tests --test-config ./test-containers-managed-api.yaml --output $(TEST_RESULTS_DIR) --namespace test-products


.PHONY: cluster/deploy
cluster/deploy: kustomize cluster/cleanup cluster/cleanup/crds cluster/prepare/crd cluster/prepare deploy/integreatly-rhmi-cr.yml
	@ - oc create -f config/rbac/service_account.yaml
	@ - cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_FORMAT}
	@ - $(KUSTOMIZE) build config/redhat-$(INSTALLATION_SHORTHAND) | oc apply -f -

.PHONY: test/unit
test/unit: export WATCH_NAMESPACE=testing-namespaces-operator
test/unit:
	@TEMPLATE_PATH=$(TEMPLATE_PATH) ./scripts/ci/unit_test.sh


.PHONY: install/olm
install/olm: cluster/cleanup/olm cluster/cleanup/crds cluster/prepare cluster/prepare/olm/subscription deploy/integreatly-rhmi-cr.yml cluster/check/operator/deployment cluster/prepare/dms cluster/prepare/pagerduty

.PHONY: test/e2e/olm
test/e2e/olm: install/olm

.PHONY: cluster/deploy/integreatly-rhmi-cr.yml
cluster/deploy/integreatly-rhmi-cr.yml: deploy/integreatly-rhmi-cr.yml
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.bootstrap.phase}' | grep -q completed, bootstrap phase, 5m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.monitoring.phase}' | grep -q completed, monitoring phase, 10m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.authentication.phase}' | grep -q completed, authentication phase, 10m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.products.phase}' | grep -q completed, products phase, 30m, 30)
ifeq ($(INSTALLATION_TYPE), managed)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.solution-explorer.phase}' | grep -q completed, solution-explorer phase, 10m, 30)
endif

.PHONY: cluster/prepare
cluster/prepare: cluster/prepare/project cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/ratelimits cluster/prepare/delorean

.PHONY: cluster/prepare/bundle
cluster/prepare/bundle: cluster/prepare/project cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/delorean

.PHONY: create/olm/bundle
create/olm/bundle:
	./scripts/bundle-rhmi-operators.sh

.PHONY: cluster/prepare/project
cluster/prepare/project:
	@ - oc new-project $(NAMESPACE)
	@oc label namespace $(NAMESPACE) monitoring-key=middleware --overwrite
	@oc project $(NAMESPACE)

.PHONY: kustomize cluster/prepare/configmaps
cluster/prepare/configmaps: kustomize
	$(KUSTOMIZE) build config/configmap | oc apply -n $(NAMESPACE) -f -

.PHONY: cluster/prepare/croaws
cluster/prepare/croaws:
	@ - oc create -f config/croaws/cro-aws-config.yml -n $(NAMESPACE)

.PHONY: cluster/prepare/crd
cluster/prepare/crd: kustomize
	$(KUSTOMIZE) build config/crd | oc apply -f -

.PHONY: cluster/prepare/local
cluster/prepare/local: kustomize cluster/prepare/project cluster/prepare/crd cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/ratelimits cluster/prepare/delorean cluster/prepare/croaws
	@ - oc create -f config/rbac/service_account.yaml -n $(NAMESPACE)
	@ - $(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc create -f -

.PHONY: cluster/prepare/olm/subscription
cluster/prepare/olm/subscription:
	oc process -p NAMESPACE=$(NAMESPACE) -f config/olm/operator-subscription-template.yml | oc create -f - -n $(NAMESPACE)
	$(call wait_command, oc get crd rhmis.integreatly.org, rhmis.integreatly.org crd, 1m, 10

.PHONY: cluster/check/operator/deployment
cluster/check/operator/deployment:
	$(call wait_command, oc get deployments rhmi-operator -n $(NAMESPACE) --output=json -o jsonpath='{.status.availableReplicas}' | grep -q 1, rhmi-operator ,2m, 10)

.PHONY: cluster/prepare/smtp
cluster/prepare/smtp:
	@-oc create secret generic $(NAMESPACE_PREFIX)smtp -n $(NAMESPACE) \
		--from-literal=host=smtp.example.com \
		--from-literal=username=dummy \
		--from-literal=password=dummy \
		--from-literal=port=587 \
		--from-literal=tls=true

.PHONY: cluster/prepare/pagerduty
cluster/prepare/pagerduty:
	@-oc create secret generic $(NAMESPACE_PREFIX)pagerduty -n $(NAMESPACE) \
		--from-literal=serviceKey=test

.PHONY: cluster/prepare/dms
cluster/prepare/dms:
	@-oc create secret generic $(NAMESPACE_PREFIX)deadmanssnitch -n $(NAMESPACE) \
		--from-literal=url=https://dms.example.com

.PHONY: cluster/prepare/ratelimits
cluster/prepare/ratelimits:
	@-oc create -n $(NAMESPACE) -f config/configmap/sku-limits-configmap.yaml

.PHONY: cluster/prepare/delorean
cluster/prepare/delorean: cluster/prepare/delorean/pullsecret

.PHONY: cluster/prepare/delorean/pullsecret
cluster/prepare/delorean/pullsecret:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
	$(MAKE) setup/service_account
	@./scripts/setup-delorean-pullsecret.sh
	$(MAKE) cluster/cleanup/serviceaccount
endif

.PHONY: cluster/cleanup
cluster/cleanup: kustomize
	@-oc delete rhmis $(INSTALLATION_NAME) -n $(NAMESPACE) --timeout=240s --wait
	@-oc delete namespace $(NAMESPACE) --timeout=60s --wait
	@-$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc delete -f -
	

.PHONY: cluster/cleanup/serviceaccount
cluster/cleanup/serviceaccount: kustomize
	@-oc delete serviceaccount ${TEMP_SERVICEACCOUNT_NAME} -n ${NAMESPACE}
	@-$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc delete -f -

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
	sed "s/INSTALLATION_NAME/$(INSTALLATION_NAME)/g" config/samples/integreatly-rhmi-cr.yaml | \
	sed "s/INSTALLATION_TYPE/$(INSTALLATION_TYPE)/g" | \
	sed "s/INSTALLATION_PREFIX/$(INSTALLATION_PREFIX)/g" | \
	sed "s/INSTALLATION_SHORTHAND/$(INSTALLATION_SHORTHAND)/g" | \
	sed "s/SELF_SIGNED_CERTS/$(SELF_SIGNED_CERTS)/g" | \
	sed "s/IN_PROW/'$(IN_PROW)'/g" | \
	sed "s/OPERATORS_IN_PRODUCT_NAMESPACE/$(OPERATORS_IN_PRODUCT_NAMESPACE)/g" | \
	sed "s/USE_CLUSTER_STORAGE/$(USE_CLUSTER_STORAGE)/g" > config/samples/integreatly-rhmi-cr.yml
	@-oc create -f config/samples/integreatly-rhmi-cr.yml

.PHONY: prepare-patch-release
prepare-patch-release:
	$(CONTAINER_ENGINE) pull quay.io/integreatly/delorean-cli:master
	$(CONTAINER_ENGINE) run --rm -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z -v "${HOME}/.delorean.yaml:/.delorean.yaml" quay.io/integreatly/delorean-cli:master delorean release openshift-ci-release --config /.delorean.yaml --olmType $(OLMTYPE) --version $(TAG)

.PHONY: release/prepare
release/prepare: kustomize
	@KUSTOMIZE_PATH=$(KUSTOMIZE) ./scripts/prepare-release.sh

.PHONY: push/csv
push/csv:
	operator-courier verify packagemanifests/$(PROJECT)
	-operator-courier push packagemanifests/$(PROJECT)/ $(REPO) $(APPLICATION_REPO) $(TAG) "$(AUTH_TOKEN)

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

.PHONY: manifest/prodsec
manifest/prodsec:
	@./scripts/prodsec-manifest-generator.sh ${TYPE_OF_MANIFEST}


# # Run tests
# ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
# test: generate fmt vet manifests
# 	mkdir -p $(ENVTEST_ASSETS_DIR)
# 	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
# 	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(TAG) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-rhmi
bundle-rhmi: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-rhmi | operator-sdk generate bundle -q --overwrite --version $(TAG) $(BUNDLE_METADATA_OPTS)

.PHONY: packagemanifests
packagemanifests: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-$(OPERATOR-TYPE) | operator-sdk generate packagemanifests --kustomize-dir=config/manifests-$(OPERATOR-TYPE) --output-dir packagemanifests/$(OPERATOR-NAME) --version $(TAG)

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
