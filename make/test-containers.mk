
.PHONY: test-containers/check/format
test-containers/check/format:
	yq v test-containers.yaml

.PHONY: test-containers/check/images
test-containers/check/images:
	yq r test-containers.yaml 'tests[*].image' | xargs -i skopeo inspect --config docker://'{}'

.PHONY: test-containers/check
test-containers/check: test-containers/check/format test-containers/check/images
