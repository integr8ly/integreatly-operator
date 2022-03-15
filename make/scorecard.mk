VERSION ?= master
IMAGE ?= quay.io/integreatly/scorecard-test-kuttl:${VERSION}

.PHONY: scorecard/build/push
scorecard/build/push:
	docker build -t ${IMAGE} -f Dockerfile.scorecard .
	docker push ${IMAGE}

.PHONY: scorecard/build
scorecard/build:
	go build -o scorecard-test-kuttl test/scorecard/main.go
