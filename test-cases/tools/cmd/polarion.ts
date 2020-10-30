import { Argv, CommandModule } from "yargs";
import { assertEpic, Jira } from "../lib/jira";
import { uploadToPolarion } from "../lib/polarion";
import { loadTestCases } from "../lib/test-case";
import { loadTestRuns } from "../lib/test-run";
import { logger } from "../lib/winston";

const POLARION_PROJECT_ID = "RedHatManagedIntegration";

interface TestCaseArgs {
    polarionUsername: string;
    polarionPassword: string;
    dumpOnly: boolean;
}
// tslint:disable:object-literal-sort-keys
const testCase: CommandModule<{}, TestCaseArgs> = {
    command: "testcase",
    describe: "Upload all test cases to Polarion",
    builder: {
        polarionUsername: {
            describe: "Jira username or set POLARION_USERNAME",
            default: process.env.POLARION_USERNAME,
            type: "string",
            demand: true,
        },
        polarionPassword: {
            describe: "Jira password or set POLARION_PASSWORD",
            default: process.env.POLARION_PASSWORD,
            type: "string",
            demand: true,
        },
        "dump-only": {
            default: false,
            type: "boolean",
        },
    },
    handler: async (args) => {
        const tests = loadTestCases();

        // Polarion Test Case Importer: https://mojo.redhat.com/docs/DOC-1075945
        //
        // prepare the testcases xml document
        const testcases = tests.map((t) => ({
            $: { id: t.id },
            title: `${t.id} - ${t.category} - ${t.title}`,
            description: t.url,
            "custom-fields": [
                {
                    "custom-field": [
                        // Level
                        { $: { content: "component", id: "caselevel" } },
                        // Component
                        { $: { content: "-", id: "casecomponent" } },
                        // Test Type
                        { $: { content: "functional", id: "testtype" } },
                        // Subtype 1
                        { $: { content: "-", id: "subtype1" } },
                        // Subtype 2
                        { $: { content: "-", id: "subtype2" } },
                        // Pos/Neg
                        { $: { content: "positive", id: "caseposneg" } },
                        // Importance
                        { $: { content: "high", id: "caseimportance" } },
                        // Automation
                        {
                            $: {
                                content: "automated",
                                id: "caseautomation",
                            },
                        },
                    ],
                },
            ],
        }));

        const document = {
            testcases: {
                $: { "project-id": POLARION_PROJECT_ID },
                properties: [
                    {
                        property: [
                            { $: { name: "lookup-method", value: "custom" } },
                        ],
                    },
                ],
                testcase: testcases,
            },
        };

        await uploadToPolarion(
            "testcase",
            document,
            args.polarionUsername,
            args.polarionPassword,
            args.dumpOnly
        );
    },
};

interface TestRunArgs {
    polarionUsername: string;
    polarionPassword: string;
    jiraUsername: string;
    jiraPassword: string;
    epic: string;
    dumpOnly: boolean;
}

const testRun: CommandModule<{}, TestRunArgs> = {
    command: "testrun",
    describe: "Report the result of all manual tests to Polarion",
    builder: {
        polarionUsername: {
            describe: "Jira username or set POLARION_USERNAME",
            default: process.env.POLARION_USERNAME,
            type: "string",
            demand: true,
        },
        polarionPassword: {
            describe: "Jira password or set POLARION_PASSWORD",
            default: process.env.POLARION_PASSWORD,
            type: "string",
            demand: true,
        },
        jiraUsername: {
            describe: "Jira username or set JIRA_USERNAME",
            default: process.env.JIRA_USERNAME,
            type: "string",
            demand: true,
        },
        jiraPassword: {
            describe: "Jira password or set JIRA_PASSWORD",
            default: process.env.JIRA_PASSWORD,
            type: "string",
            demand: true,
        },
        epic: {
            describe: "the key of the epic containing all manual tests",
            type: "string",
            demand: true,
        },
        "dump-only": {
            default: false,
            type: "boolean",
        },
    },
    handler: async (args) => {
        const jira = new Jira(args.jiraUsername, args.jiraPassword);

        const epic = await jira.findIssue(args.epic);
        assertEpic(epic);

        const runs = await loadTestRuns(jira, `"Epic Link" = ${epic.key}`);

        const testcases = runs
            .filter((r) => r.result !== "Skipped")
            .map((r) => {
                const testcase: any = {
                    $: { name: r.title },
                    properties: {
                        property: {
                            $: { name: "polarion-testcase-id", value: r.id },
                        },
                    },
                };

                switch (r.result) {
                    case "Failed":
                        testcase.failure = { $: { message: r.link } };
                        break;
                    case "Blocked":
                        testcase.error = { $: { message: r.link } };
                        break;
                    case "Passed":
                        // Do nothing
                        break;
                }

                return testcase;
            });

        const document = {
            testsuites: {
                properties: {
                    property: [
                        {
                            $: {
                                name: "polarion-project-id",
                                value: POLARION_PROJECT_ID,
                            },
                        },
                        {
                            $: {
                                name: "polarion-testrun-title",
                                value: epic.fields.summary,
                            },
                        },
                        {
                            $: {
                                name: "polarion-lookup-method",
                                value: "custom",
                            },
                        },
                    ],
                },
                testsuite: {
                    $: { tests: 1 },
                    testcase: testcases,
                },
            },
        };

        await uploadToPolarion(
            "xunit",
            document,
            args.polarionUsername,
            args.polarionPassword,
            args.dumpOnly
        );

        logger.warn(
            "Remember to set the Planned In version to the created Test Run"
        );
    },
};

const polarion: CommandModule = {
    command: "polarion",
    describe: "upload test cases and test runs to Polarion",
    builder: (args: Argv): Argv => {
        return args.command(testCase).command(testRun);
    },
    handler: () => {
        // nothing
    },
};

export { polarion };
