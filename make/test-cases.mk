
.PHONY: test-cases/lint
test-cases/lint:
	@cd test-cases/ && npm install
	@cd test-cases/ && npm run lint
