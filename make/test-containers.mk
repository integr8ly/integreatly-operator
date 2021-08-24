
.PHONY: test-containers/check/format
test-containers/check/format:
	yq e 'true' test-containers.yaml > /dev/null

.PHONY: test-containers/check/images
test-containers/check/images:
	yq e  '.tests[].image' test-containers.yaml | xargs -i skopeo inspect --config docker://'{}'

.PHONY: test-containers/check
test-containers/check: test-containers/check/format test-containers/check/images
