# E2E tests

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