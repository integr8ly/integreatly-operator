# Integreatly Test Cases

- Test cases are located in the directory `tests/` and are organized in subdirectories by categories.
- Each markdown file `*.md` inside the `tests/` represents one test case.
- Each test case must start with the ID and a title immediately after the metadata `# ID - This is a title`.
- The ID must be unique across all test cases and have the same format `[CATEGORY-LETTER][TWO-DIGIT-NUMBER]`. The only exception to this format is when a single test case can be performed on more than one product (e.g. RHOAM and RHMI), but some test case steps differ (e.g. test case mentions components that are not common for both products). In that case, it is ideal to split the test case in 2 files (each file for a single product) and add the suffix `A` or `B` to the test ID in those files: `[CATEGORY-LETTER][TWO-DIGIT-NUMBER][A-or-B]`
- The file name of each test case must match the title of the test case without special characters or spaces. Use `./tools.sh rename` to fix all file names.
- The `./tools.sh` script requires Nodejs >= 10.
- Try to write test cases using the standard syntax described in the [test-template.md](./fixtures/test-template.md).
- General guidelines in `common/general-guidelines.md` are imported on the bottom of every test case.

## Index

- [How to create a test case](#How-to-create-a-test-case)
- [How to include a manual test case in the next release](#How-to-include-a-manual-test-case-in-the-next-release)
- [How to bulk update the target version on the test cases](#How-to-bulk-update-the-target-version-on-the-test-cases)
- [How to estimate a test case](#How-to-estimate-a-test-case)
- [How to automate a test case and link it back](#How-to-automate-a-test-case-and-link-it-back)
- [How to create Jira tasks for the manual tests](#How-to-create-Jira-tasks-for-the-manual-tests)
- [List and export the test cases](#List-and-export-the-test-cases)
- [How to upload all test case to Polarion](#How-to-upload-all-test-case-to-Polarion)
- [How to report the results of the manual tests to Polarion](#How-to-report-the-results-of-the-manual-tests-to-Polarion)
- [Test Case Metadata](#Test-Case-Metadata)
- [Prettier](#Prettier)
- [How to export the test cases to CSV](#How-to-export-the-test-cases-to-CSV)

## How to create a test case

Copy the test template to a category inside the `tests/` directory:

```
cp fixtures/test-template.md tests/somecategory/some-test.md
```

and edit the test case following the template structure.

> If you are adding an automated test there is no need to fill
> the **Prerequisites** and **Steps** sections

Once your done ensure the file name and the markdown format are correct:

```bash
# to verify the test cases
npm run lint

# to fix the name
npm run rename

# to fix the format
npm run prettier
```

Commit everything and open a new PR.

## How to include a manual test case in the next release

> Attention: all changes need to be manually pushed to master or to the release branch

By default we don't execute all manual tests on each release, therefore if we know we have made a change to a component and a specific test should be executed to verify the change or prevent regressions we must manually add the test case to the next release.

To do that the next version needs to be added to the `targets`, for example if the last released version was `2.7.0` then the next release version would be `2.8.0`.

```
---
targets:
  - 2.8.0
---

# Z00 - Verify
```

If need to update multiple test cases by component read: [How to bulk update the target version on the test cases](#how-to-bulk-update-the-target-version-on-the-test-cases)

## How to bulk update the target version on the test cases

> Attention: all changes are applied locally and need to be manually pushed to master or to the release branch

To automatically add the target version to all test cases with a target version older than 3 releases, use the following command:

```
./tools.sh plan release --product PRODUCT_NAME --target TARGET_VERSION
```

> For example if we set the TARGET_VERSION to `2.8.0` than all test cases which latest target version for the same major release is equal or minor to `2.5.0` will receive the target `2.8.0` and therefore included in the `2.8.0` release

To automatically add the target version to all test cases with a specific component:

```
./tools.sh plan for --product PRODUCT_NAME --target TARGET_VERSION --component COMPONENT
```

## How to estimate a test case

Each manual test case should have a rough estimation of the time (in hours) required to manually complete it.

The estimation should be set in the test case metadata like this

```
---
estimate: 2h
---

# Z00 - My test
```

- Use this scale for the estimation: 15m, 30m 1h, 2h, 3h, 5h, 8h, 13h
- If the estimated tests is bigger than 8h than it should be split
- When estimating the test do not count the time of reporting bugs, or debugging issues
- Try to estimate the test as someone that is doing it for the first time

## How to automate a test case and link it back

1. Create the automated test in the appropriate test suite depending on the type and complexity of the test.

   **Test suites:**

   - [Functional Test Suite](https://github.com/integr8ly/integreatly-operator/tree/master/test/functional)

2. The title of the automated test must contain the ID of the test case `ID - Title`.

   > If you need/want to split the test case in multiple smaller tests when automating it that's
   > completely fine but we need to track them back here, therefore we need to create a new
   > test cases with a new ID for each automated test so that each automated test can be still
   > linked to a test case in this repo.

3. The content/steps of the automated test must match the manual test

   > If the automated test does not completely cover the manual test then the test case should be
   > split and the part that is not automated should become a new manual test

4. Once the automated test is working and running as part of the nightly pipeline the related test case should be:

   - Flag the test cases as automated by setting the `automated` tag
   - Add the link to to the automated test in the test case

Example: [MR!63](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases/merge_requests/63)

## How to create Jira tasks for the manual tests

Prerequisites:

- Nodejs >= 10

> Always refer to the [2.X Release Testing Workflow](https://github.com/RHCloudServices/integreatly-help/blob/master/qe-guides/2.x-release-testing-workflow.md) on how to create the Jira tasks during the release testing

To crate the Jira tasks for the test cases you need first to create an Epic in Jira with the `fixVersion` that you want to target. Only test cases with the same target version or marked as `per-release` or `per-build` will be created in the Epic.

To see the list of test cases that will be created in the Epic you can use the `export` cmd:

```bash
./tools.sh export csv --product PRODUCT_NAME --target VERSION --environment ENVIRONMENT | column -t -s,
```

Use the `jira` cmd to create the Jira tasks for the test cases and add them to the Epic

```bash
JIRA_USERNAME=yourusername JIRA_PASSWORD=yourpassword ./tools.sh jira --epic EPICKEY-00 --product PRODUCT_NAME --environment ENVIRONMENT
```

If you need to link the new tasks to the task of a previous test round use the `previous-epic` option:

```bash
JIRA_USERNAME=yourusername@redhat.com JIRA_PASSWORD=yourpassword ./tools.sh jira --epic EPICKEY-01 --previous-epic EPICKEY-00 --product PRODUCT_NAME
```

> The `previous-epic` option will link each new task to the task in the previous epic with the same ID and it
> will set the Priority of the new task depending on the Resolution of the previous task.
>
> Resolution -> Priority:
>
> - Rejected -> Blocker
> - Deferred -> Critical
> - [New Test] -> Major
> - Won't Do -> Minor
> - Done -> Optional
>
> And automatically close as **Won't Do** all tests marked as **Won't Do** or **Done** in the previous Epic

It is also possible to set the Jira username and password in environment variables:

```
 ./tools.sh jira --epic EPICKEY-00 --product PRODUCT_NAME
```

## List and export the test cases

Use the `export csv` cmd to list and export the test cases in csv:

```
./tools.sh export csv --product PRODUCT_NAME
```

use the `--output` option to save it to file

```
./tools.sh export csv --product PRODUCT_NAME --output /tmp/alltests.csv
```

to export only specific test cases use the `--filter` option:

```
./tools.sh export csv --product PRODUCT_NAME --filter components=product-3scale tags=^automated
```

to pretty print the csv output on the terminal:

```
./tools.sh export csv --product PRODUCT_NAME | column -t -s, | less -S
```

## How to upload all test case to Polarion

This command will automatically upload all test cases to Polarion:

```bash
POLARION_USERNAME=yourusername POLARION_PASSWORD=yourpassword ./tools.sh polarion testcase --product PRODUCT_NAME
```

## How to report the results of the manual tests to Polarion

> Attention: before doing this ensure to have uploaded all test cases to Polarion

This command will read all test results from Jira, and upload them to Polarion.

For parameter `template`, use the required template id from `Test Runs -> Manage Templates`

```bash
JIRA_USERNAME=ju JIRA_PASSWORD=jp POLARION_USERNAME=pu POLARION_PASSWORD=pp ./tools.sh polarion testrun --epic INTLY-5390 --product PRODUCT_NAME --template v1_0_0_rc1
```

## Test Case Metadata

### Products

This field specifies the list of products the test case can be executed against.
Each product must contain the fields `name` and `environments`. It can also contain a field `targets`. See below for more details about these fields.

#### Product names:

- `rhmi` - Red Hat Managed Integration
- `rhoam` - Red Hat OpenShift API Management

### Environments

The environment field specify against which environment/setup the test need to be executed.

- `osd-post-upgrade` This is the environment where most of the tests will be executed and also the **default** choice for most of the tests. It consist in a BYOC OSD on AWS cluster installed with the previous version using addon-flow and upgraded to the version to test.
- `osd-fresh-install` This environment should be used for tests that are for sure not affected from the upgrade. It consist in a BYOC OSD on AWS cluster installed with the version to test using the addon-flow.
- `osd-private-post-upgrade` This is a special environment and should be used only for tests that are specifically targeting it, the only difference with the pervious environments, is that this environment resides behind a VPN
- `rhpds` This is the RHMI demo environment and should be used only for tests that are specifically targeting it.
- `external` This is a special tag that is used to identify tests that needs a special environment or a stand-alone one, the test in this case would have to specify how to setup the cluster.

### Targets

Targets version defines against which release the test case is going to be executed next time. This is useful to include or exclude the test cases form a specific release.

All test cases need to define at least a `targets` version or set the `per-release`, `per-build` or `manual-selection` tag.

In this example the test case `Z00` would be included in the `2.7.0` and `2.9.0` releases but excluded from any other release:

```
---
targets:
  - 2.7.0
  - 2.9.0
---

# Z00 - Verify
```

### Tags

Tags are used to define specific characteristics of a test case.

- `automated` Indicates that this test case is already automated and that it doesn't need to be executed manually
- `per-build` Indicates that this test case should be executed on each build even if passed in the previous one
- `per-release` Indicates that this test case should be executed on each release but can be skip if passed in the previous build
- `manual-selection` Indicates that this test case is excluded from the standard test recycle and therefore it will not be retested every tree releases
- `destructive` Indicates that the test case may conflict with other tests and therefore can't be executed in parallel with other tests, all destructive tests must be executed sequentially

E.g.:

```
---
tags:
  - per-build
---

# Z00 - Verify
```

### Components

The component field is used to decide which tests should be added in the next release.

- `product-3scale` 3Scale minor or major product upgrades
- `product-amq` AMQ minor or major product upgrades
- `product-fuse` Fuse minor or major product upgrades
- `product-apicurito` Apicurito minor or major product upgrades
- `product-codeready` CodeReady minor or major product upgrades
- `product-sso` SSO minor or major product upgrades
- `product-ups` UPS minor or major product upgrades
- `product-data-sync` Data Sync minor or major product upgrades
- `monitoring` Monitoring stack changes or upgrade

### Automation Jiras

As new manual test cases are being added, there should also be corresponding automation tasks for them in JIRA. Each manual test case for which automation task exist should have `automation` in its metadata pointing to the automation jira, e.g.:

```
---
automation:
  - INTLY-7421
---

# Z00 - Verify
```

## Prettier

> All of the test cases must be prettified before being committed

You can prettify the test cases using the command line:

```bash
npm run prettier
```

Or using the VS Code extension: https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode

## How to export the test cases to CSV

Run the `./tools.sh export csv` script to export all test cases in a CSV table:

```
./tools.sh export csv --product PRODUCT_NAME --output testcases.csv
```
