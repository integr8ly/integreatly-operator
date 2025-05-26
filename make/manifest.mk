IMAGE_MAPPINGS?=$(shell sh -c "find manifests/ -name image_mirror_mapping")

.PHONY: manifest/check/image_mirror_mapping
manifest/check/image_mirror_mapping:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
		$(error found image_mirror_mapping in $(IMAGE_MAPPINGS))
else
		@echo "No image_mirror_mapping files found in manifests directory"
endif

.PHONY: manifest/check/registries
manifest/check/registries:
	@! grep "registry.stage.redhat.io" -r manifests/ --include=*clusterserviceversion.{yml,yaml}
	@! grep "quay.io/integreatly/delorean" -r manifests/ --include=*clusterserviceversion.{yml,yaml}
	@! grep "registry-proxy.engineering.redhat.com" -r manifests/ --include=*clusterserviceversion.{yml,yaml}

.PHONY: manifest/check/graph
manifest/check/graph:
	delorean ews check-olm-graph -d ./manifests

.PHONY: manifest/check
manifest/check: manifest/check/image_mirror_mapping
