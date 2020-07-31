
.PHONY: prow/e2e/credentials
prow/e2e/credentials:
	$(CONTAINER_ENGINE) pull quay.io/integreatly/delorean-cli:master
	$(CONTAINER_ENGINE) run -it --rm -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z quay.io/integreatly/delorean-cli:master e2e-test-extract-creds.sh $(CI_NAMESPACE)


.PHONY: prow/e2e/tail
prow/e2e/tail: prow/e2e/credentials
	oc -n $(CI_NAMESPACE) logs -f e2e -c test --tail=50
