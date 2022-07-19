# Tests

### Unit tests

Running unit tests:
```sh
make test/unit
```

### E2E tests

If you have RHOAM operator installed using cluster storage (`useClusterStorage: true`), all [AWS tests](https://github.com/integr8ly/integreatly-operator/blob/27c4a8c4fdf3461247fad2bb20fe958d4b709a99/test/functional/tests.go#L6-L12) are being skipped (because all AWS tests would fail).
To override this, you can provide an env var `BYPASS_STORAGE_TYPE_CHECK=true`.

To run E2E tests against a clean OpenShift cluster using operator-sdk, build and push an image 
to your own quay repo, then run the command below changing the installation type based on which type you are testing:
```
make test/e2e INSTALLATION_TYPE=<managed/managed-api/multitenant-managed-api> OPERATOR_IMAGE=<your/repo/image:tag>
```

To run E2E tests against an existing RHMI cluster:
```
make test/functional
```

To run a single E2E test against a running cluster run the command below where E03 is the start of the test description:
```
INSTALLATION_TYPE=<managed/managed-api/multitenant-managed-api> TEST=E03 make test/e2e/single
```
### Product tests

To run products tests against an existing RHMI cluster:
```
make test/products/local
```

### Scorecard tests

For testing all scorecard tests within this repo, it is recommended to have RHOAM installed (or RHOAM install triggered) on an OSD/OCP cluster:

1. To deploy RHOAM operator on a cluster (using master branch image) and install RHOAM:
```bash
export INSTALLATION_TYPE=managed-api
make cluster/deploy
```
2. Prepare scorecard tests

```bash
make scorecard/bundle/prepare
make scorecard/service_account/prepare
```
3. Run scorecard test
```bash
# To run a specific test, set SCORECARD_TEST_NAME env var with a name
# of the test taken from '.labels.test' field in bundle/tests/scorecard/config.yaml
# Example:
make scorecard/test/run SCORECARD_TEST_NAME=basic-check-spec-test
```

**Note**

If you are doing some changes to the code of the scorecard test image (`Dockerfile.scorecard`, files in `test/scorecard` folder), you can test them by creating a new image in your quay.io account

```bash
export IMAGE=quay.io/<YOUR-ACCOUNT-ID>/scorecard-test-kuttl:<your-tag>
make scorecard/build/push IMAGE=$IMAGE
```

And update the `image` field in the test you want to run in `bundle/tests/scorecard/config.yaml` file to point to your image in quay.io
