
.PHONY: test-cases/lint
test-cases/lint:
	cd test-cases/ && npm install
	cd test-cases/ && npm run lint || \
		(echo -e "\nRun 'make test-cases/fix' to automatically fix all lint issues\n"; exit 23)

.PHONY: test-cases/fix
test-cases/fix:
	cd test-cases/ && npm install
	cd test-cases/ && npm run prettier
	cd test-cases/ && npm run rename
