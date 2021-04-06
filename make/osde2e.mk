DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
MANAGED_API_TEST_HARNESS_IMAGE ?= $(REG)/$(ORG)/integreatly-operator-test-harness:osde2e

.PHONY: image/osde2e/build
image/osde2e/build:
	go mod vendor
	docker build . -f Dockerfile.osde2e -t $(MANAGED_API_TEST_HARNESS_IMAGE)

.PHONY: image/osde2e/push
image/osde2e/push:
	docker push $(MANAGED_API_TEST_HARNESS_IMAGE)

.PHONY: image/osde2e/build/push
image/osde2e/build/push: image/osde2e/build image/osde2e/push

.PHONY: test/compile/osde2e
test/compile/osde2e:
	CGO_ENABLED=0 go test -v -c -o managed-api-test-harness.test ./test/osde2e
