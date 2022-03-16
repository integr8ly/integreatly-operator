VERSION ?= master
IMAGE ?= quay.io/integreatly/scorecard-test-kuttl:${VERSION}

SCORECARD_NAMESPACE ?= redhat-rhoam-operator
SCORECARD_SA_NAME ?= rhoam-test-runner

SCORECARD_TEST_NAME ?= ""
SCORECARD_SKIP_CLEANUP ?= true

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
	operator-sdk scorecard ./bundle --namespace=${SCORECARD_NAMESPACE} --output json --selector=test=${SCORECARD_TEST_NAME} --skip-cleanup=${SCORECARD_SKIP_CLEANUP}

.PHONY: scorecard/build/push
scorecard/build/push:
	docker build -t ${IMAGE} -f Dockerfile.scorecard .
	docker push ${IMAGE}

.PHONY: scorecard/build
scorecard/build:
	go build -o scorecard-test-kuttl test/scorecard/main.go
