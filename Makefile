include ./make/*.mk

ORG ?= integreatly

CONFIG_IMAGE ?= 'quay.io/integreatly/managed-api-service-config:latest'

REG=quay.io
SHELL=/bin/bash

PKG=github.com/integr8ly/integreatly-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=integreatly-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
AUTH_TOKEN=$(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "$(QUAY_USERNAME)", "password": "$(QUAY_PASSWORD)"}}' | jq -r '.token')
CREDENTIALS_MODE=$(shell oc get cloudcredential cluster -o json | jq -r ".spec.credentialsMode")
TEMPLATE_PATH="$(shell pwd)/templates/monitoring"
IN_PROW ?= "false"
SKIP_FINAL_DB_SNAPSHOTS ?= "true"
# DEV_QUOTA value is the default QUOTA when install locally and is per 100,000
# acceptable values are
# if 10 then 1M
# if 50 then 5M
# if 100 then 10M
# if 200 then 20M
# if 500 then 50M
# if 1 then 100k
DEV_QUOTA ?= "1"
CUSTOM_DOMAIN ?= ''
SMTP_USER  ?= ''
SMTP_ADDRESS ?= ''
SMTP_PASS ?= ''
SMTP_PORT ?= ''
SMTP_FROM ?= ''
MAINTENANCE_DAY ?= ''
MAINTENANCE_HOUR ?= ''
CRO_ROLE_ARN ?= 'arn:aws:iam::123456789012:role/example'
THREESCALE_ROLE_ARN ?= 'arn:aws:iam::123456789012:role/example'

TYPE_OF_MANIFEST ?= master

CONTAINER_ENGINE ?= docker
CONTAINER_PLATFORM ?= linux/amd64
TEST_RESULTS_DIR ?= test-results
TEMP_SERVICEACCOUNT_NAME=rhmi-operator
SANDBOX_NAMESPACE ?= sandbox-rhoam-operator
PROJECT_ROOT := $(shell git rev-parse --show-toplevel)

# These tags are modified by the prepare-release script.
RHMI_TAG ?= 2.9.0
RHOAM_TAG ?= 1.44.0

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
export INSTALLATION_TYPE ?= managed-api
export CLUSTER_CONFIG ?= redhat-rhoam

export ALERT_SMTP_FROM ?= noreply-alert@devshift.org
export USE_CLUSTER_STORAGE ?= true
export OPERATORS_IN_PRODUCT_NAMESPACE ?= false # e2e tests and createInstallationCR() need to be updated when default is changed
export DELOREAN_PULL_SECRET_NAME ?= integreatly-delorean-pull-secret
export ALERTING_EMAIL_ADDRESS ?= noreply-test@rhmi-redhat.com
export BU_ALERTING_EMAIL_ADDRESS ?= noreply-test@rhmi-redhat.com

ifeq ($(shell test -e envs/$(INSTALLATION_TYPE).env && echo -n yes),yes)
	include envs/$(INSTALLATION_TYPE).env
endif

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

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(TAG) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
    BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: moq
moq: $(MOQ) ## Download moq locally if necessary.
$(MOQ): $(LOCALBIN)
	$(call go-install-tool,$(MOQ),github.com/matryer/moq,$(MOQ_VERSION))
	@[ -f "$(MOQ)" ] || ln -sf $(MOQ)-$(MOQ_VERSION) $(MOQ)

.PHONY: setup/service_account/oc_login
setup/service_account/oc_login:
	@oc login --token=$(shell sh -c "oc create token rhmi-operator -n ${NAMESPACE} --duration=24h") --server=$(shell sh -c "oc whoami --show-server") --kubeconfig=TMP_SA_KUBECONFIG --insecure-skip-tls-verify=true

.PHONY: setup/service_account
setup/service_account: kustomize
	@-oc new-project $(NAMESPACE)
	@oc project $(NAMESPACE)
	@-oc create -f config/rbac/service_account.yaml -n $(NAMESPACE)
	@$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc replace --force -f -
	$(MAKE) setup/service_account/oc_login

.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

.PHONY: install/sandboxrhoam/operator
install/sandboxrhoam/operator:
	@-oc new-project $(SANDBOX_NAMESPACE)
	@-oc process -p RHOAM_NAMESPACE=$(SANDBOX_NAMESPACE) -f config/developer-sandbox/sandbox-operator-template.yml | oc create -f - -n $(SANDBOX_NAMESPACE)

.PHONY: install/sandboxrhoam/config
install/sandboxrhoam/config:
	@-oc process -p RHOAM_OPERATOR_NAMESPACE=$(SANDBOX_NAMESPACE) -f config/developer-sandbox/sandbox-config-template.yml | oc create -f - -n $(SANDBOX_NAMESPACE)
	@oc label namespace $(SANDBOX_NAMESPACE) monitoring-key=middleware --overwrite
	@-oc process -f config/developer-sandbox/sandbox-rhoam-quickstart.yml | oc create -f -

.PHONY: code/run
code/run: code/gen cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty setup/service_account
	@KUBECONFIG=TMP_SA_KUBECONFIG WATCH_NAMESPACE=$(NAMESPACE) QUOTA=$(DEV_QUOTA) go run ./cmd/main.go

.PHONY: code/rerun
code/rerun: setup/service_account
	@KUBECONFIG=TMP_SA_KUBECONFIG WATCH_NAMESPACE=$(NAMESPACE) go run ./cmd/main.go

.PHONY: code/run/service_account
code/run/service_account: code/run

.PHONY: code/run/delorean
code/run/delorean: cluster/cleanup cluster/prepare cluster/prepare/local deploy/integreatly-rhmi-cr.yml code/run/service_account

.PHONY: code/compile
code/compile: code/gen
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/main.go

pkg/api/integreatly/v1alpha1/zz_generated.openapi.go: api/v1alpha1/rhmi_types.go
	$(OPENAPI_GEN) --logtostderr=true -o "" \
		-i ./api/v1alpha1/ \
		-p ./api/v1alpha1/ \
		-O zz_generated.openapi \
		-h ./hack/boilerplate.go.txt \
		-r "-"

api/integreatly/v1alpha1/zz_generated.deepcopy.go: controller-gen api/v1alpha1/rhmi_types.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: code/gen
code/gen: $(MOQ) api/integreatly/v1alpha1/zz_generated.deepcopy.go
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@echo "Ensuring moq is available at $(MOQ)"
	@if [ ! -f "$(MOQ)" ]; then \
		echo "moq not found, installing directly..." ; \
		mkdir -p $(LOCALBIN) ; \
		GOBIN=$(LOCALBIN) go install github.com/matryer/moq@$(MOQ_VERSION) ; \
	fi
	@chmod +x $(MOQ)
	@mkdir -p /tmp/bin && ln -sf $(MOQ) /tmp/bin/moq
	@PATH="/tmp/bin:$(LOCALBIN):$$PATH" go generate ./...
	mv ./config/crd/bases/integreatly.org_apimanagementtenants.yaml ./config/crd-sandbox/bases

.PHONY: code/check
code/check: golangci-lint
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)
	GOLANGCI_LINT_CACHE=/tmp/.cache $(GOLANGCI_LINT) run ./...
	go vet ./...


.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/gen
	echo "build image $(OPERATOR_IMAGE)"
	$(CONTAINER_ENGINE) build --platform=$(CONTAINER_PLATFORM) -t ${OPERATOR_IMAGE} .

.PHONY: image/push
image/push:
	echo "push image $(OPERATOR_IMAGE)"
	$(CONTAINER_ENGINE) push $(OPERATOR_IMAGE)

.PHONY: image/build/push
image/build/push: image/build image/push


############ E2E TEST COMMANDS ############
.PHONY: test/e2e/rhoam/prow
test/e2e/rhoam/prow: export component := integreatly-operator
test/e2e/rhoam/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/rhoam/prow: export INSTALLATION_TYPE := managed-api
test/e2e/rhoam/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/rhoam/prow: export WATCH_NAMESPACE := redhat-rhoam-operator
test/e2e/rhoam/prow: export NAMESPACE_PREFIX := redhat-rhoam-
test/e2e/rhoam/prow: export NAMESPACE:= $(NAMESPACE_PREFIX)operator
test/e2e/rhoam/prow: export INSTALLATION_PREFIX := redhat-rhoam
test/e2e/rhoam/prow: export INSTALLATION_NAME := rhoam
test/e2e/rhoam/prow: export INSTALLATION_SHORTHAND := rhoam
test/e2e/rhoam/prow: export IN_PROW = "true"
test/e2e/rhoam/prow: test/e2e

.PHONY: test/e2e/multitenant-rhoam/prow
test/e2e/multitenant-rhoam/prow: export CLUSTER_CONFIG:=redhat-sandbox
test/e2e/multitenant-rhoam/prow: export component := integreatly-operator
test/e2e/multitenant-rhoam/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/multitenant-rhoam/prow: export INSTALLATION_TYPE := multitenant-managed-api
test/e2e/multitenant-rhoam/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/multitenant-rhoam/prow: export WATCH_NAMESPACE := sandbox-rhoam-operator
test/e2e/multitenant-rhoam/prow: export NAMESPACE_PREFIX := sandbox-rhoam-
test/e2e/multitenant-rhoam/prow: export NAMESPACE:= $(NAMESPACE_PREFIX)operator
test/e2e/multitenant-rhoam/prow: export INSTALLATION_PREFIX := sandbox-rhoam
test/e2e/multitenant-rhoam/prow: export INSTALLATION_NAME := rhoam
test/e2e/multitenant-rhoam/prow: export INSTALLATION_SHORTHAND := sandbox
test/e2e/multitenant-rhoam/prow: export IN_PROW = "true"
test/e2e/multitenant-rhoam/prow: test/e2e

.PHONY: test/e2e
test/e2e: export SURF_DEBUG_HEADERS=1
test/e2e: cluster/deploy/e2e test/prepare/ocp/obo
	cd test && go clean -testcache && go test -v ./e2e -timeout=120m -ginkgo.v

.PHONY: test/e2e/single
test/e2e/single: export WATCH_NAMESPACE := $(NAMESPACE)
test/e2e/single: 
	cd test && go clean -testcache && go test ./functional -ginkgo.focus="$(TEST).*" -test.v -ginkgo.v -ginkgo.progress -timeout=80m

.PHONY: test/functional
test/functional: export WATCH_NAMESPACE := $(NAMESPACE)
test/functional:
	# Run the functional tests against an existing cluster. Make sure you have logged in to the cluster.
	cd test && go clean -testcache && go test -v ./functional -timeout=120m

.PHONY: test/osde2e
test/osde2e: export WATCH_NAMESPACE := $(NAMESPACE)
test/osde2e: export SKIP_FLAKES := $(SKIP_FLAKES)
test/osde2e:
	# Run the osde2e tests against an existing cluster. Make sure you have logged in to the cluster.
	cd test && go clean -testcache && go test ./osde2e -test.v -ginkgo.v -ginkgo.progress -timeout=120m

.PHONY: test/prepare/ocp/obo
test/prepare/ocp/obo:
	# We need to apply these CRDs and create the -observability project on OCP clusters in order to install RHOAM with OBO.
	# The PrometheusRules CRDS will be removed when Phase 2 of the OBO migration is complete.
	@oc apply -f https://raw.githubusercontent.com/rhobs/observability-operator/main/bundle/manifests/monitoring.rhobs_prometheusrules.yaml
	@ - oc new-project $(NAMESPACE)-observability
	@oc label namespace $(NAMESPACE)-observability monitoring-key=middleware openshift.io/cluster-monitoring="true" --overwrite

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

.PHONY: cluster/deploy
cluster/deploy: kustomize cluster/cleanup cluster/cleanup/crds cluster/prepare/crd cluster/prepare cluster/prepare/rbac/dedicated-admins deploy/integreatly-rhmi-cr.yml install-oo
	@ - oc create -f config/rbac/service_account.yaml
	@ - cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_FORMAT}
	@ - $(KUSTOMIZE) build config/$(CLUSTER_CONFIG) | oc apply -f -

.PHONY: cluster/deploy/e2e
cluster/deploy/e2e: kustomize cluster/cleanup cluster/cleanup/crds cluster/prepare/crd cluster/prepare cluster/prepare/rbac/dedicated-admins deploy/integreatly-rhmi-cr.yml
	@ - oc create -f config/rbac/service_account.yaml
	@ - cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_FORMAT}
	@ - $(KUSTOMIZE) build config/$(CLUSTER_CONFIG) | oc apply -f -

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
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.installation.phase}' | grep -q completed, installation phase, 40m, 30)

.PHONY: cluster/prepare
cluster/prepare: cluster/prepare/project cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/pagerduty cluster/prepare/delorean cluster/prepare/addon-params cluster/prepare/addon-instance

.PHONY: cluster/prepare/bundle
cluster/prepare/bundle: cluster/prepare/project cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/delorean

.PHONY: create/olm/bundle
create/olm/bundle:
	./scripts/bundle-rhmi-operators.sh

.PHONY: create/3scale/index
create/3scale/index:
	PRODUCT=3scale ./scripts/create-product-index.sh

.PHONY: create/marin3r/index
create/marin3r/index:
	PRODUCT=marin3r ./scripts/create-product-index.sh

.PHONY: create/cloud-resource-operator/index
create/cloud-resource-operator/index:
	PRODUCT=cloud-resource ./scripts/create-product-index.sh

.PHONY: create/rhsso/index
create/rhsso/index:
	PRODUCT=rhsso ./scripts/create-product-index.sh

.PHONY: cluster/prepare/project
cluster/prepare/project:
	@ - oc new-project $(NAMESPACE_PREFIX)cloud-resources-operator
	@oc label namespace $(NAMESPACE_PREFIX)cloud-resources-operator monitoring-key=middleware openshift.io/cluster-monitoring="true" --overwrite
	@ - oc new-project $(NAMESPACE_PREFIX)3scale
	@oc label namespace $(NAMESPACE_PREFIX)3scale monitoring-key=middleware openshift.io/cluster-monitoring="true" --overwrite
	@ - oc new-project $(NAMESPACE)
	@oc label namespace $(NAMESPACE) monitoring-key=middleware openshift.io/cluster-monitoring="true" --overwrite
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
	$(KUSTOMIZE) build config/crd-sandbox | oc apply -f -

.PHONY: cluster/prepare/local
cluster/prepare/local: kustomize cluster/prepare/project cluster/prepare/crd cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/addon-params cluster/prepare/delorean cluster/prepare/rbac/dedicated-admins cluster/prepare/addon-instance cluster/prepare/rhoam-config install-oo
	@if [ "$(CREDENTIALS_MODE)" = Manual ]; then \
		echo "manual mode (sts)"; \
		$(MAKE) cluster/prepare/sts; \
	else \
	  	echo "mint mode"; \
	fi \

	@ - oc create -f config/rbac/service_account.yaml -n $(NAMESPACE)
	@ - $(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc create -f -

.PHONY: cluster/prepare/olm/subscription
cluster/prepare/olm/subscription:
	oc process -p NAMESPACE=$(NAMESPACE) -f config/olm/operator-subscription-template.yml | oc create -f - -n $(NAMESPACE)
	$(call wait_command, oc get crd rhmis.integreatly.org, rhmis.integreatly.org crd, 1m, 10)

.PHONY: cluster/check/operator/deployment
cluster/check/operator/deployment:
	$(call wait_command, oc get deployments rhmi-operator -n $(NAMESPACE) --output=json -o jsonpath='{.status.availableReplicas}' | grep -q 1, rhmi-operator ,2m, 10)

.PHONY: cluster/prepare/smtp
cluster/prepare/smtp:
	@-oc create secret generic $(NAMESPACE_PREFIX)smtp -n $(NAMESPACE) \
		--from-literal=host= \
		--from-literal=username= \
		--from-literal=password= \
		--from-literal=port= \
		--from-literal=tls=

.PHONY: cluster/prepare/pagerduty
cluster/prepare/pagerduty:
	@-oc create secret generic $(NAMESPACE_PREFIX)pagerduty -n $(NAMESPACE) \
		--from-literal=serviceKey=test

.PHONY: cluster/prepare/dms
cluster/prepare/dms:
	@-oc create secret generic $(NAMESPACE_PREFIX)deadmanssnitch -n $(NAMESPACE) \
		--from-literal=url=https://dms.example.com

.PHONY: cluster/prepare/addon-params
cluster/prepare/addon-params:
	@-oc process -n $(NAMESPACE) QUOTA=$(DEV_QUOTA) DOMAIN=$(CUSTOM_DOMAIN) \
 		USERNAME=$(SMTP_USER) HOST=$(SMTP_ADDRESS) PASSWORD=$(SMTP_PASS) PORT=$(SMTP_PORT) FROM=$(SMTP_FROM) DAY=$(MAINTENANCE_DAY) HOUR=$(MAINTENANCE_HOUR) -f config/secrets/addon-params-secret.yaml | oc apply -f -

.PHONY: cluster/prepare/sts
cluster/prepare/sts:
	@-oc process -n $(NAMESPACE_PREFIX)cloud-resources-operator NAME=sts-credentials NAMESPACE=$(NAMESPACE_PREFIX)cloud-resources-operator ROLE_ARN=$(CRO_ROLE_ARN) -f config/secrets/sts-secret.yaml | oc apply -f -
	@-oc process -n $(NAMESPACE_PREFIX)3scale NAME=sts-s3-credentials NAMESPACE=$(NAMESPACE_PREFIX)3scale ROLE_ARN=$(THREESCALE_ROLE_ARN) -f config/secrets/sts-secret.yaml | oc apply -f -


.PHONY: cluster/prepare/addon-instance
cluster/prepare/addon-instance:
	@-oc apply -f config/samples/addoninstance_v1alpha1.yaml

.PHONY: cluster/prepare/quota/trial
cluster/prepare/quota/trial:
	@-oc process -n $(NAMESPACE) -f config/secrets/quota-trial-secret.yaml | oc apply -f -

.PHONY: cluster/prepare/delorean
cluster/prepare/delorean: cluster/prepare/delorean/pullsecret

.PHONY: cluster/prepare/delorean/pullsecret
cluster/prepare/delorean/pullsecret:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
	$(MAKE) setup/service_account
	./scripts/setup-delorean-pullsecret.sh
	$(MAKE) cluster/cleanup/serviceaccount
endif

.PHONY:cluster/prepare/rbac/dedicated-admins
cluster/prepare/rbac/dedicated-admins:
	@-oc create -f config/rbac/dedicated_admins_rbac.yaml

.PHONY: cluster/prepare/rhoam-config
cluster/prepare/rhoam-config:
ifeq ($(IN_PROW),true)
	@echo "Not creating rhoam-config ClusterPackage because IN_PROW is set to true"
else
	@-oc process -n $(NAMESPACE) CONFIG_IMAGE=$(CONFIG_IMAGE) NAMESPACE=$(NAMESPACE) -f config/hive-config/package.yaml | oc apply -f -
endif

.PHONY: cluster/cleanup
cluster/cleanup: export WATCH_NAMESPACE := $(NAMESPACE)
cluster/cleanup: kustomize
	@-oc delete clusterpackage $(OLM_TYPE) --wait
	@-oc delete rhmis $(INSTALLATION_NAME) -n $(NAMESPACE) --timeout=240s --wait
	@-oc delete namespace $(NAMESPACE) --timeout=60s --wait
	@-oc delete namespace $(NAMESPACE_PREFIX)cloud-resources-operator --timeout=60s --wait
	@-oc delete namespace $(NAMESPACE_PREFIX)3scale --timeout=60s --wait
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
	@-oc delete crd rhmis.integreatly.org
	@-oc delete crd apimanagementtenants.integreatly.org

.PHONY: cluster/cleanup/rbac/dedicated-admins
cluster/cleanup/rbac/dedicated-admins:
	@-oc delete -f config/rbac/dedicated_admins_rbac.yaml

.PHONY: deploy/integreatly-rhmi-cr.yml
deploy/integreatly-rhmi-cr.yml:
	@echo "selfSignedCerts = $(SELF_SIGNED_CERTS)"
	sed "s/INSTALLATION_NAME/$(INSTALLATION_NAME)/g" config/samples/integreatly-rhmi-cr.yaml | \
	sed "s/INSTALLATION_TYPE/$(INSTALLATION_TYPE)/g" | \
	sed "s/INSTALLATION_PREFIX/$(INSTALLATION_PREFIX)/g" | \
	sed "s/INSTALLATION_SHORTHAND/$(INSTALLATION_SHORTHAND)/g" | \
	sed "s/SELF_SIGNED_CERTS/$(SELF_SIGNED_CERTS)/g" | \
	sed "s/OPERATORS_IN_PRODUCT_NAMESPACE/$(OPERATORS_IN_PRODUCT_NAMESPACE)/g" | \
	sed "s/USE_CLUSTER_STORAGE/$(USE_CLUSTER_STORAGE)/g" > config/samples/integreatly-rhmi-cr.yml
	# in_prow annotation is used to allow for installation on small Prow cluster and might be used to skip tests failing in Prow
	yq e -i '.metadata.annotations.in_prow="IN_PROW"' config/samples/integreatly-rhmi-cr.yml
	$(SED_INLINE) "s/IN_PROW/'$(IN_PROW)'/g" config/samples/integreatly-rhmi-cr.yml
	# skip_final_db_snapshots annotation specifies if CRO should skip creating final AWS snapshots of the postgres and redis DBs before deleting them during RHOAM uninstallation
	yq e -i '.metadata.annotations.skip_final_db_snapshots="SKIP_FINAL_DB_SNAPSHOTS"' config/samples/integreatly-rhmi-cr.yml
	$(SED_INLINE) "s/SKIP_FINAL_DB_SNAPSHOTS/'$(SKIP_FINAL_DB_SNAPSHOTS)'/g" config/samples/integreatly-rhmi-cr.yml

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
	-operator-courier push packagemanifests/$(PROJECT)/ $(REPO) $(APPLICATION_REPO) $(TAG) "$(AUTH_TOKEN)"

.PHONY: gen/push/csv
gen/push/csv: release/prepare push/csv

.PHONY: vendor/check
vendor/check: vendor/fix
	git diff --exit-code vendor/
	git diff --exit-code go.sum
	git diff --exit-code test/go.sum

.PHONY: vendor/check/prow
vendor/check/prow:
	sh scripts/setup-private-git-access.sh
	make vendor/check

.PHONY: vendor/fix
vendor/fix:
	go mod tidy
	go mod vendor
	# Vendor test module
	# entire command needs to be inline as cd runs in a subshell and would not run in the directory if not inline
	cd test; go mod tidy; go mod vendor

.PHONY: manifest/prodsec
manifest/prodsec:
	@./scripts/prodsec-manifest-generator.sh ${TYPE_OF_MANIFEST}

.PHONY: kubebuilder/check
kubebuilder/check: code/gen
	git diff --exit-code config/crd/bases
	git diff --exit-code config/rbac/role.yaml

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	$(OPERATOR_SDK) generate kustomize manifests --interactive=false -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-rhoam
bundle-rhoam: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-rhoam | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) --output-dir ./bundles/managed-api-service/$(TAG) --kustomize-dir config/manifests-rhoam

.PHONY: packagemanifests
packagemanifests: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-$(OPERATOR-TYPE) | $(OPERATOR_SDK) generate packagemanifests --kustomize-dir=config/manifests-$(OPERATOR-TYPE) --output-dir packagemanifests/$(OPERATOR-NAME) --version $(TAG)

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	$(CONTAINER_ENGINE) build --platform=$(CONTAINER_PLATFORM) -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# USAGE: make olm/bundle BUNDLE_TAG="quay.io/mstoklus/integreatly-index:1.15.2" VERSION=1.15.2 OLM_TYPE=managed-api-service will build a bundle from 1.15.2 bundles/managed-api-service directory.
.PHONY: olm/bundle
olm/bundle:
	$(CONTAINER_ENGINE) build --platform=$(CONTAINER_PLATFORM) -f bundles/$(OLM_TYPE)/bundle.Dockerfile -t $(BUNDLE_TAG) --build-arg version=$(VERSION) .

.PHONY: coverage
coverage:
	hack/codecov.sh

.PHONY: commits/check
commits/check:
	@./scripts/commits-check.sh

GOSEC_BIN := $(PWD)/bin/gosec
GOSEC_VERSION := 2.22.7

.PHONY: install-gosec
install-gosec:
	@mkdir -p $(PWD)/bin
	curl -sSfL https://github.com/securego/gosec/releases/download/v$(GOSEC_VERSION)/gosec_$(GOSEC_VERSION)_linux_amd64.tar.gz \
	  | tar -xz -C $(PWD)/bin gosec


.PHONY: gosec
gosec: install-gosec
	# Module layout causes issues if not using go workspace but is not supported in Cachito for now
	# https://github.com/securego/gosec/issues/682
	rm -rf /tmp/gosec-scan && mkdir -p /tmp/gosec-scan/src/app
	cp -r . /tmp/gosec-scan/src/app
	cd /tmp/gosec-scan/src/app && \
	  GO111MODULE=on $(GOSEC_BIN) -exclude-dir test ./...


##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
MOQ ?= $(LOCALBIN)/moq

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.3
CONTROLLER_TOOLS_VERSION ?= v0.16.1
ENVTEST_VERSION ?= release-0.19
MOQ_VERSION ?= v0.5.3
GOLANGCI_LINT_VERSION ?= v1.64.2

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
	set -e; \
	package=$(2)@$(3) ;\
	echo "Downloading $${package}" ;\
	rm -f $(1) || true ;\
	GOBIN=$(LOCALBIN) GOFLAGS= go install $${package} ;\
	mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@if [ ! -f $(KUSTOMIZE) ]; then \
		echo "Downloading kustomize $(KUSTOMIZE_VERSION) binary..."; \
		curl -s -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$(KUSTOMIZE_VERSION)/kustomize_$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz" \
		  | tar -xz -C $(LOCALBIN) kustomize; \
	fi

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	@if [ ! -f $(GOLANGCI_LINT) ]; then \
		echo "Downloading golangci-lint $(GOLANGCI_LINT_VERSION) binary..."; \
		curl -sSfL https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION:v%=%)-linux-amd64.tar.gz \
		  | tar -xz -C $(LOCALBIN) --strip-components=1 golangci-lint-$(GOLANGCI_LINT_VERSION:v%=%)-linux-amd64/golangci-lint; \
	fi



.PHONY: mkdocs/serve
mkdocs/serve:
	mkdocs serve

.PHONY: test/unit/prometheus
test/unit/prometheus:
	@find prometheus-unit-testing/tests -type f | xargs promtool test rules

.PHONY: test/unit/prometheus/single
test/unit/prometheus/single:
	@promtool test rules $(PROM_TEST_RULE_FILE)

.PHONY: test/lint
test/lint: golangci-lint
	@cd $(PROJECT_ROOT) && $(GOLANGCI_LINT) run

.PHONY: test/scripts
test/scripts:
	# Preform a basic check on scripts checking for a non zero exit
	echo "Running make release/prepare"
	SEMVER=$(RHOAM_TAG) make release/prepare

.PHONY: install-oo
install-oo:
	@echo "Creating Cluster Observability Operator Subscription"
	kubectl apply -f  config/samples/subscription_oo.yaml

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

.PHONY: build-installer-rhoam
build-installer-rhoam: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml
