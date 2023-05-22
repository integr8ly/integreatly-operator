DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
INTEGREATLY_OPERATOR_TEST_HARNESS_IMAGE ?= $(REG)/$(ORG)/integreatly-operator-test-harness:latest
INTEGREATLY_OPERATOR_TEST_EXTERNAL_IMAGE ?= $(REG)/$(ORG)/integreatly-operator-test-external:latest

.PHONY: image/functional/build
image/functional/build:
	go mod vendor
	$(CONTAINER_ENGINE) build --platform=$(CONTAINER_PLATFORM) . -f Dockerfile.functional -t $(INTEGREATLY_OPERATOR_TEST_HARNESS_IMAGE)

.PHONY: image/external/build
image/external/build:
	go mod vendor
	$(CONTAINER_ENGINE) build --platform=$(CONTAINER_PLATFORM) . -f Dockerfile.external -t $(INTEGREATLY_OPERATOR_TEST_EXTERNAL_IMAGE)

.PHONY: image/external/push
image/external/push:
	$(CONTAINER_ENGINE) push $(INTEGREATLY_OPERATOR_TEST_EXTERNAL_IMAGE)


.PHONY: image/functional/push
image/functional/push:
	$(CONTAINER_ENGINE) push $(INTEGREATLY_OPERATOR_TEST_HARNESS_IMAGE)

.PHONY: image/functional/build/push
image/functional/build/push: image/functional/build image/functional/push

.PHONY: test/compile/functional
test/compile/functional:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -mod=readonly -v -c -o integreatly-operator-test-harness.test ./test/functional
