# Scorecard tests

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