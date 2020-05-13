SHELL=/bin/bash

.PHONY: delorean/cluster/prepare
delorean/cluster/prepare:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
	@echo Detected a delorean ews branch. The integreatly-delorean-secret.yml is required.
	@echo Please contact the delorean team to get this if you do not already have it.
	@echo Add it to the root dir of this repo and rerun the desired target if the target fails on it not existing
	@./scripts/setup-delorean-pullsecret.sh
endif
