ORG=integreatly
NAMESPACE=integreatly
PROJECT=integreatly-operator
REG=quay.io
SHELL=/bin/bash
TAG=v1.8.4
PKG=github.com/integr8ly/integreatly-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=integreatly-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)

.PHONY: setup/dep
setup/dep:
	@echo Installing dep
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	@echo setup complete

.PHONY: setup/moq
setup/moq:
	dep ensure
	cd vendor/github.com/matryer/moq/ && go install .

.PHONY: setup/dedicated
setup/dedicated:
	cd ./scripts && ./dedicated-setup.sh

.PHONY: clean/dedicated
clean/dedicated:
	cd ./scripts && ./dedicated-cleanup.sh

.PHONY: setup/travis
setup/travis:
	@echo Installing Operator SDK
	@curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v0.8.1/operator-sdk-v0.8.1-x86_64-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/

.PHONY: setup/service_account
setup/service_account:
	@oc replace --force -f deploy/role.yaml -n $(NAMESPACE)
	@oc replace --force -f deploy/service_account.yaml -n $(NAMESPACE)
	@oc replace --force -f deploy/role_binding.yaml -n $(NAMESPACE)
	@oc replace --force -f deploy/clusterrole.yaml -n $(NAMESPACE)
	@oc replace --force -f deploy/cluster_role_binding.yaml -n $(NAMESPACE)

.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

.PHONY: code/run
code/run:
	@operator-sdk up local --namespace=$(NAMESPACE)

.PHONY: code/run/service_account
code/run/service_account: setup/service_account
	@oc login --token=$(shell oc serviceaccounts get-token integreatly-operator -n ${NAMESPACE})
	@operator-sdk up local --namespace=$(NAMESPACE)

.PHONY: code/compile
code/compile:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/manager

.PHONY: code/gen
code/gen:
	operator-sdk generate k8s
	@go generate ./...

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/compile
	@operator-sdk build $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: image/push
image/push:
	docker push $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: image/build/test
image/build/test:
	operator-sdk build --enable-tests $(REG)/$(ORG)/$(PROJECT):$(TAG)

.PHONY: test/unit
test/unit:
	@echo Running tests:
	go test -v -race -coverprofile=coverage.out ./pkg/...

.PHONY: test/e2e
test/e2e:
	kubectl apply -f deploy/test-e2e-pod.yaml -n $(PROJECT)
	$(SHELL) ./scripts/stream-pod ${TEST_POD_NAME} $(PROJECT)

.PHONY: cluster/prepare
cluster/prepare:
	-oc create -f https://raw.githubusercontent.com/integr8ly/manifests/master/operator-source.yml
	@oc new-project $(NAMESPACE)
	@oc project $(NAMESPACE)
	@oc process -f deploy/s3-secrets.yaml \
		-p INSTALLATION_NAMESPACE=$(NAMESPACE) \
		-p AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-p AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		-p AWS_BUCKET=$(AWS_BUCKET) \
		-p AWS_REGION=eu-west-1 | oc apply -f -
	@oc create secret generic github-oauth-secret \
		--from-literal=clientId=$(GH_CLIENT_ID) \
		--from-literal=secret=$(GH_CLIENT_SECRET)

.PHONY: cluster/prepare/local
cluster/prepare/local: cluster/prepare
	-oc create -f deploy/crds/*.crd.yaml
	@oc create -f deploy/service_account.yaml
	@oc create -f deploy/role.yaml
	@oc create -f deploy/role_binding.yaml