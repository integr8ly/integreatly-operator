DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
INTEGREATLY_OPERATOR_TEST_HARNESS_IMAGE ?= $(REG)/$(ORG)/integreatly-operator-test-harness:latest

.PHONY: image/functional/build
image/functional/build:
	go mod vendor
	docker build . -f Dockerfile.functional -t $(INTEGREATLY_OPERATOR_TEST_HARNESS_IMAGE)

.PHONY: image/functional/push
image/functional/push:
	docker push $(INTEGREATLY_OPERATOR_TEST_HARNESS_IMAGE)

.PHONY: image/functional/build/push
image/functional/build/push: image/functional/build image/functional/push

.PHONY: test/compile/functional
test/compile/functional:
	CGO_ENABLED=0 go test -v -c -o integreatly-operator-test-harness.test ./test/functional
