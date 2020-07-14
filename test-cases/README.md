# Integreatly Test Cases

- Test cases are located in the directory `tests/` and are organized in subdirectories by categories.
- Each markdown file `*.md` inside the `tests/` represents one test case.
- Each test case must start with the ID and a title immediately after the metadata `# ID - This is a title`.
- The ID must be unique across all test cases and have the same format `[CATEGORY-LETTER][TWO-DIGIT-NUMBER]`.
- The file name of each test case must match the title of the test case without special characters or spaces. Use `./tools.sh rename` to fix all file names.
- The `./tools.sh` script requires Nodejs >= 10.
- Try to write test cases using the standard syntax described in the [test-template.md](./fixtures/test-template.md).
- General guidelines in `common/general-guidelines.md` are imported on the bottom of every test case.

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
# to fix the name
npm run rename

# to fix the format
npm run prettier
```

Commit everything and open a new MR.

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

4. Once the automated test is working and running as part of the nightly pipeline the related test
   case should be:

   - Flag the test cases as automated by setting the `automated` tag
   - Add the link to to the automated test in the test case

Example: [MR!63](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases/merge_requests/63)

## How to create Jira tasks for the manual tests

Prerequisites:

- Nodejs >= 10

> Always refer to the [2.X Release Testing Workflow](https://github.com/RHCloudServices/integreatly-help/blob/master/qe-guides/2.x-release-testing-workflow.md) on how to create the Jirat tasks during the release testing

> Use the `--help` flags to get the full documentation of all avialble flags ([Command-line interface Built-in usage help](https://en.wikipedia.org/wiki/Command-line_interface#Built-in_usage_help))
>
> ```bash
> /tools.sh jira --help
> ```

To create a tasks for each test case in in Jira under an epic use:

```bash
./tools.sh jira --jira-username yourusername --jira-password yourpassword --epic EPICKEY-00
```

If you need to link the new tasks to the task of a previous test round use the `previousEpic` option:

```bash
./tools.sh jira --jira-username yourusername --jira-password yourpassword --epic EPICKEY-01 --previousEpic EPICKEY-00
```

> The `previousEpic` option will link each new task to the task in the previous epic with the same ID and it
> will set the Priority of the new task depending on the Resolution of the previous task.
>
> Resolution -> Priority:
>
> - Rejected -> Blocker
> - Deferred -> Critical
> - [New Test] -> Major
> - Won't Do -> Minor
> - Done -> Optional

To automatically close as **Won't Do** all tests marked as **Won't Do** or **Done** in the previous Epic use the `--auto-resolve` option:

```bash
./tools.sh jira --epic EPICKEY-01 --previousEpic EPICKEY-00 --auto-resolve
```

It is also possible to set the Jira username and password in environment variables:

```
JIRA_USERNAME=yourusername JIRA_PASSWORD=yourpassword ./tools.sh jira --epic EPICKEY-00
```

To create only some test cases you can use the option `--filter` to filter
the test cases by tags:

```bash
./tools.sh jira --jira-username yourusername --jira-password yourpassword --epic EPICKEY-00 --filter per-build,^automated
```

this will create a task for each test case with the tag `per-build` and not with the tag `automate`.
`^` stands for **not**. Multiple tags can be combined using `,` but they will be always in **and** relation.

## How to upload all test case to Polarion

This command will automatically upload all test cases to Polarion:

```bash
POLARION_USERNAME=yourusername POLARION_PASSWORD=yourpassword ./tools.sh polarion testcase
```

## How to report the results of the manual tests to Polarion

> Attention: before doing this ensure to have uploaded all test cases to Polarion

This command will read all test results from Jira, and upload them to Polarion.

```bash
JIRA_USERNAME=ju JIRA_PASSWORD=jp POLARION_USERNAME=pu POLARION_PASSWORD=pp ./tools.sh polarion testrun --epic INTLY-5390
```

## Test Case Metadata

### Tags

Tags are used to define specific behaviors for the test case.

- `automated` Indicates that this test case is already automated and that it doesn't need to be executed manually
- `per-build` Indicates that this test case should be executed on each build even if passed in the previous one
- `per-release` Automatically generated if the `targets` field is set, and it indicates that the tests will be executed on each release

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
- `product-codeready` CodeReady minor or major product upgrades
- `product-sso` SSO minor or major product upgrades
- `product-ups` UPS minor or major product upgrades
- `product-data-sync` Data Sync minor or major product upgrades
- `monitoring` Monitoring stack changes or upgrade
- `openshift` Special component using for testing OpenShift upgrades

### Targets

The Targets define against which release the test case would have to be executed the next time. This is useful to include or exclude the test cases form a specific release.

If the Targets is not defined then the tests will be marked as per-release and would be included in all releases.

In this example the test case `Z00` would be included in the `2.7.0` and `2.9.0` releases but excluded from any other release:

```
---
targets:
  - 2.7.0
  - 2.9.0
---

# Z00 - Verify
```

### Automation Jiras

As new manual test cases are being added, there should also be corresponding automation tasks for them in JIRA. Each manual test case for which automation task exist should have `automation_jiras` in its metadata pointing to the automation jira, e.g.:

```
---
automation_jiras:
  - INTLY-7421
---

# Z00 - Verify
```

## Prettier

> All of the test cases must be prettify before being committed

You can prettify the test cases using the command line:

```bash
npm run prettier
```

Or using the VS Code extension: https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode

## How to export the test cases to CSV

Run the `./tools.sh export csv` script to export all test cases in a CSV table:

```
./tools.sh export csv --output testcases.csv
```
