IMAGE_MAPPINGS?=$(shell sh -c "find manifests/ -name image_mirror_mapping")

.PHONY: manifest/check/image_mirror_mapping
manifest/check/image_mirror_mapping:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
		$(error found image_mirror_mapping in $(IMAGE_MAPPINGS))
else
		@echo "No image_mirror_mapping files found in manifests directory"
endif

.PHONY: manifest/check
manifest/check: manifest/check/image_mirror_mapping