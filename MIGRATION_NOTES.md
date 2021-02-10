# Migration notes
## To operator-sdk 1.2.0

### Steps performed

1. Scaffolded new project using operator-sdk CLI
2. Copied `RHMI` and `RHMIConfig` types to new project
3. Bottom-up copying of packages and fixing dependencies and imports
    1. `resources` package
    2. `products` package
    3. `controllers` package. The controller implementation changes drastically in operator-sdk 1.x, so the logic is copied manually in the scaffolded controllers

### Known issues

The following issues have arised during the migration

1. #### `k8sutil` package not found
    operator-sdk made the `k8sutil` package internal, and removed some of the logic that is used across the operator.

    The following error message is shown as dependencies use the `k8sutil` package:
    ```
    github.com/operator-framework/operator-sdk/pkg/k8sutil: module github.com/operator-framework/operator-sdk@latest found (v1.2.0), but does not contain package github.com/operator-framework/operator-sdk/pkg/k8sutil
    ```

    Asked about it in Slack:
    > https://coreos.slack.com/archives/C3VS0LV41/p1608040300275700
    

### Next steps

* We stopped using `context` (We pass always `context.TODO()` when needed). Newly scaffolded controllers use
  `context.Background()`. Might be a good time to check if it's a good practice to use it and won't result
  in memory leaks.
* Marin3r needs to be upgraded to 0.6.0, as previous versions use packages from operator-sdk that are replaced in 1.x
    > The dependency has been updated to 0.6.0 but the operator manifests will need to be updated too
* CRO needs to be upgraded as it references the `k8sutil` package
* Research webhook changes

    operator-sdk and OLM now support webhooks. We implemented our custom webhook logic but it should be migrated to the supported approach

* e2e tests need to be migrated
    > Operator SDK 1.0.0+ removes support for the legacy test framework and no longer supports the operator-sdk test subcommand. All affected tests should be migrated to use envtest.
    > 
    > https://sdk.operatorframework.io/docs/building-operators/golang/migration/#migrate-your-tests

* Add vendor folder
* Check code generation (CRDs and DeepCopy functions)
* Integrate `make` commands into new Makefile