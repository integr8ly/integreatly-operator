VERSION ?= master
IMAGE ?= quay.io/integreatly/scorecard-test-kuttl:${VERSION}

SCORECARD_NAMESPACE ?= redhat-rhoam-operator
SCORECARD_SA_NAME ?= rhoam-test-runner

SCORECARD_TEST_NAME ?= ""
SCORECARD_SKIP_CLEANUP ?= true
ARTIFACTS_DIR ?= "logs/artifacts"
TEST_TIMEOUT ?= "3000s"

.PHONY: scorecard/bundle/prepare
scorecard/bundle/prepare:
	$(KUSTOMIZE) build config/manifests-rhoam | operator-sdk generate bundle --kustomize-dir=config/manifests-rhoam
	@ cp -r config/scorecard/kuttl bundle/tests/scorecard

.PHONY: scorecard/service_account/prepare
scorecard/service_account/prepare:
	@ - oc create serviceaccount ${SCORECARD_SA_NAME} -n ${SCORECARD_NAMESPACE}
	@ - oc create clusterrolebinding default-sa-crb --clusterrole=cluster-admin --serviceaccount="${SCORECARD_NAMESPACE}":"${SCORECARD_SA_NAME}"

.PHONY: scorecard/service_account/cleanup
scorecard/service_account/cleanup:
	@ - oc delete serviceaccount ${SCORECARD_SA_NAME} -n ${SCORECARD_NAMESPACE}
	@ - oc delete clusterrolebinding default-sa-crb

.PHONY: scorecard/test/run
scorecard/test/run:
	operator-sdk scorecard ./bundle --namespace=${SCORECARD_NAMESPACE} --output json --service-account ${SCORECARD_SA_NAME}  --wait-time ${TEST_TIMEOUT} --test-output ${ARTIFACTS_DIR} --selector=test=${SCORECARD_TEST_NAME} --skip-cleanup=${SCORECARD_SKIP_CLEANUP}

.PHONY: scorecard/build/push
scorecard/build/push: scorecard/build
	$(CONTAINER_ENGINE) push ${IMAGE}

.PHONY: scorecard/build
scorecard/build:
	$(CONTAINER_ENGINE) build --platform=$(CONTAINER_PLATFORM) -t ${IMAGE} -f Dockerfile.scorecard .

.PHONY: scorecard/compile
scorecard/compile:
	go build -mod=readonly -o scorecard-test-kuttl test/scorecard/main.go
