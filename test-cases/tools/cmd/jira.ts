import * as markdown2confluence from "markdown2confluence-cws";
import { CommandModule } from "yargs";
import { assertEpic, Issue, Jira } from "../lib/jira";
import {
    isDestructive,
    isPerBuild,
    loadTestCases,
    releaseFilter,
    TestCase,
} from "../lib/test-case";
import { loadTestRuns, TestRun } from "../lib/test-run";
import { logger } from "../lib/winston";

const GENERAL_GUIDELINES_URL =
    "https://github.com/integr8ly/integreatly-operator/tree/master/test-cases/common/general-guidelines.md";

function appendLinkToGeneralGuidelines(content: string): string {
    const guidelines = `## General guidelines for testing\n${GENERAL_GUIDELINES_URL}`;
    return content.concat("\n", guidelines);
}

function extractSprintId(sprintInfo: string[] | number): number {
    if (sprintInfo != null) {
        const found = /(id\=)(\d+)/.exec(sprintInfo[0]);
        if (found) {
            return parseInt(found[2], 10);
        }
    }
    return null;
}

function prependOriginLink(
    content: string,
    file: string,
    link: string
): string {
    return `**Origin:** [${file}](${link})\n\n${content}`;
}

function toPriority(run?: TestRun) {
    if (!run) {
        return "Major"; // without a previous run
    }

    switch (run.result) {
        case "Failed":
            return "Blocker";
        case "Blocked":
            return "Critical";
        case "Passed":
            return "Normal";
        case "Skipped":
            return "Minor";
    }
}

function toIssue(
    test: TestCase,
    epicKey: string,
    projectKey: string,
    fixVersionId: string,
    fixBuildId: string,
    team: string,
    sprint: number,
    priority: string,
    security: string
): Issue {
    let content = prependOriginLink(test.content, test.file, test.url);

    content = appendLinkToGeneralGuidelines(content);

    let title = `${test.category} - ${test.title}`;
    if (isDestructive(test)) {
        title = `[DESTRUCTIVE] - ${title}`;
    }
    title = `${test.id} - ${title}`;

    return {
        fields: {
            versions: [{ id: fixVersionId }],
            assignee: null,
            components: [{ name: "Testing" }],
            customfield_12311140: epicKey,
            customfield_12312442: { id: fixBuildId },
            customfield_12313240: team,
            customfield_12310940: sprint,
            description: markdown2confluence(content),
            fixVersions: [{ id: fixVersionId }],
            issuetype: { name: "Task" },
            labels: ["test-case"],
            priority: { name: priority },
            project: { key: projectKey },
            summary: title,
            security: { name: security },
        },
    };
}

function toIssueLink(inward: Issue, outward: TestRun) {
    return {
        inwardIssue: { key: inward.key },
        outwardIssue: { key: outward.issue.key },
        type: { name: "Sequence" },
    };
}

function toIssueBlock(inward: Issue, outward: Issue) {
    return {
        inwardIssue: { key: inward.key },
        outwardIssue: { key: outward.key },
        type: { name: "Blocks" },
    };
}

interface Args {
    jiraToken: string;
    epic: string;
    previousEpic?: string;
    environment: string;
    product: string;
    dryRun: boolean;
}

// tslint:disable:object-literal-sort-keys
const jira: CommandModule<{}, Args> = {
    command: "jira",
    describe: "create Jira task for each test case",
    builder: {
        jiraToken: {
            describe: "Jira token or set JIRA_TOKEN",
            default: process.env.JIRA_TOKEN,
            type: "string",
            demand: true,
        },
        environment: {
            demand: true,
            describe: "the environment name used to filter out the test cases",
            type: "string",
        },
        product: {
            demand: true,
            describe: "the product name used to filter out the test cases",
            type: "string",
        },
        epic: {
            demand: true,
            describe: "key of the epic to use as parent of all new tasks",
            type: "string",
        },
        "previous-epic": {
            describe: "link the new taks to a previous epic",
            type: "string",
        },
        "dry-run": {
            describe: "print test cases that will be create",
            type: "boolean",
            default: false,
        },
    },
    handler: async (args) => {
        const jiraApi = new Jira(args.jiraToken);

        const epic = await jiraApi.findIssue(args.epic);
        assertEpic(epic);

        const fixVersion = epic.fields.fixVersions[0];
        if (!fixVersion) {
            throw new Error(
                `the epic ${args.epic} does not have a Fix Version`
            );
        }

        const fixBuild = epic.fields.customfield_12312442;
        if (!fixBuild) {
            throw new Error(`the epic ${args.epic} does not have a Fix Build`);
        }

        const team = epic.fields.customfield_12313240;

        const sprintId = extractSprintId(epic.fields.customfield_12310940);

        const security = "Red Hat Employee";

        let previousRuns: TestRun[] = [];

        if (args.previousEpic) {
            const previousEpic = await jiraApi.findIssue(args.previousEpic);
            assertEpic(previousEpic);

            previousRuns = await loadTestRuns(
                jiraApi,
                `"Epic Link"  = ${previousEpic.key}`
            );
        }

        const project = epic.fields.project.key;

        let tests = loadTestCases(args.product);
        tests = releaseFilter(tests, args.environment, fixVersion.name);

        let hasDestructive = false;
        let firstDestructive = null;
        let lastDestructive = null;
        if (tests.filter((x) => isDestructive(x)).length) {
            hasDestructive = true;
            tests.sort((x, y) =>
                isDestructive(x) === isDestructive(y)
                    ? 0
                    : isDestructive(x)
                    ? -1
                    : 1
            );
        }

        for (const test of tests) {
            const previousRun = previousRuns.find((run) => run.id === test.id);

            const issue = toIssue(
                test,
                args.epic,
                project,
                fixVersion.id,
                fixBuild.id,
                team,
                sprintId,
                toPriority(previousRun),
                security
            );

            if (args.dryRun) {
                logger.info(
                    `will create task: '${issue.fields.summary}' in project '${issue.fields.project.key}'`
                );
            } else {
                const result = await jiraApi.addNewIssue(issue);
                logger.info(
                    `created task '${result.key}' '${issue.fields.summary}'`
                );

                if (hasDestructive) {
                    if (isDestructive(test)) {
                        if (test === tests[0]) {
                            lastDestructive = result;
                            firstDestructive = result;
                        } else {
                            await jiraApi.issueLink(
                                toIssueBlock(lastDestructive, result)
                            );
                            logger.info(
                                `'${result.key}' blocked by '${lastDestructive.key}'`
                            );
                            lastDestructive = result;
                        }
                    } else {
                        await jiraApi.issueLink(
                            toIssueBlock(result, firstDestructive)
                        );
                        logger.info(
                            `'${firstDestructive.key}' blocked by '${result.key}'`
                        );
                    }
                }

                if (previousRun) {
                    await jiraApi.issueLink(toIssueLink(result, previousRun));
                    logger.info(`   linked to '${previousRun.issue.key}'`);

                    if (
                        !isPerBuild(test) &&
                        (previousRun.result === "Passed" ||
                            previousRun.result === "Skipped")
                    ) {
                        await jiraApi.resolveIssue(result.key);
                        logger.info(`   automatically resolved as "Won't Do"`);
                    }
                }
            }
        }
    },
};

export { jira };
