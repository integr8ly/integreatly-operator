TYPES_FILE?=$(shell sh -c "cat apis/v1alpha1/rhmi_types.go")

.PHONY: versions/check
versions/check:
ifneq ( ,$(findstring CHANGEME,$(TYPES_FILE)))
		$(error found CHANGEME in /apis/v1alpha1/rhmi_types.go)
else
		@echo "CHANGEME string not found in rhmi_types file"
endif