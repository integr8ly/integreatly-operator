---
environments:
  - osd-post-upgrade
targets:
  - 2.7.0
---

# N10 - Verify upgrade by simulated new version

## Description

Verify that an upgrade to the simulated next version succeeds. This test case is based on [INTLY-9492](https://issues.redhat.com/browse/INTLY-9492).

## Steps

1. Create a simulated 2.6.0 release. Inside the root directory of the integreatly operator (make sure proper RC tag is used) run:
   `SEMVER=2.8.0 ./scripts/prepare-release.sh`
2. This will create a 2.8.0 directory inside deploy/olm-catalog/integreatly-operator. `cd` into this directory and create, build and push the bundle:
   `opm alpha bundle generate -d . --channels alpha --package integreatly --output-dir bundle --default alpha`
   `docker build -f bundle.Dockerfile -t quay.io/<your quay user>/integreatly-bundle:2.8.0 .`
   `docker push quay.io/<your quay user>/integreatly-bundle:2.8.0 .`
3. Now create an index containing this new bundle:
   `opm index add --bundles quay.io/<your quay user>/integreatly-bundle:2.8.0 --build-tool docker --tag quay.io/<your quay user>/integreatly-index:2.8.0`
4. Push the index:
   `docker push quay.io/<your quay user>/integreatly-index:2.8.0`
5. Create the following catalogsource on the cluster:

```
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: rhmi-operators
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/<your quay user>/integreatly-index:2.8.0
```

6. See [N02 Upgrade RHMI](https://github.com/integr8ly/integreatly-operator/blob/master/test-cases/tests/upgrade/n02-upgrade-rhmi.md) step 2. on how to edit RHMIConfig CR to trigger the upgrade.

7. Once the upgrade is finished, the version in RHMI CR should be 2.8.0.
