import * as markdown2confluence from "markdown2confluence-cws";
import { CommandModule } from "yargs";
import { assertEpic, Issue, Jira } from "../lib/jira";
import { filterTests, loadTestCases, TestCase } from "../lib/test-case";
import { loadTestFiles } from "../lib/test-file";
import { loadTestRuns, TestRun } from "../lib/test-run";
import { flat } from "../lib/utils";
import { logger } from "../lib/winston";

const GENERAL_GUIDELINES_URL =
    "https://github.com/integr8ly/integreatly-operator/tree/master/test-cases/common/general-guidelines.md";

function appendLinkToGeneralGuidelines(content: string): string {
    const guidelines = `## General guidelines for testing\n${GENERAL_GUIDELINES_URL}`;
    return content.concat("\n", guidelines);
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
            return "Optional";
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
    priority: string
): Issue {
    let content = prependOriginLink(
        test.content,
        test.file.file,
        test.file.link
    );

    content = appendLinkToGeneralGuidelines(content);

    const title = `${test.id} - ${test.category} - ${test.title}`;

    return {
        fields: {
            assignee: null,
            components: [{ name: "Testing" }],
            customfield_12311140: epicKey,
            customfield_12312442: { id: fixBuildId },
            description: markdown2confluence(content),
            fixVersions: [{ id: fixVersionId }],
            issuetype: { name: "Task" },
            labels: ["test-case"],
            priority: { name: priority },
            project: { key: projectKey },
            summary: title
        }
    };
}

function toIssueLink(run: TestRun) {
    return {
        outwardIssue: { key: run.issue.key },
        type: { name: "Sequence" }
    };
}

function toBlockedByLink(run: string) {
    return {
        outwardIssue: { key: run },
        type: { name: "Dependency" }
    };
}

interface Args {
    jiraUsername: string;
    jiraPassword: string;
    epic: string;
    previousEpic?: string;
    filter?: string;
    dryRun: boolean;
    autoResolve: boolean;
}

// tslint:disable:object-literal-sort-keys
const jira: CommandModule<{}, Args> = {
    command: "jira",
    describe: "create Jira task for each test case",
    builder: {
        jiraUsername: {
            demand: true,
            default: process.env.JIRA_USERNAME,
            describe: "Jira username or set JIRA_USERNAME",
            type: "string"
        },
        jiraPassword: {
            demand: true,
            default: process.env.JIRA_PASSWORD,
            describe: "Jira password or set JIRA_PASSWORD",
            type: "string"
        },
        filter: {
            describe: "filter test to create by tags",
            type: "string"
        },
        epic: {
            demand: true,
            describe: "key of the epic to use as parent of all new tasks",
            type: "string"
        },
        previousEpic: {
            describe: "link the new taks to a previous epic",
            type: "string"
        },
        "dry-run": {
            describe: "print test cases that will be create",
            type: "boolean",
            default: false
        },
        "auto-resolve": {
            describe:
                "tasks that passed [Done] or were skipped [Won't Do] in previous epic will be resolved (Requires --previousEpic)",
            type: "boolean",
            default: false
        }
    },
    handler: async args => {
        if (!args.previousEpic && args.autoResolve) {
            throw new Error(
                "--auto-resolve can only be used when a previous epic is included"
            );
        }

        let tests = flat(loadTestFiles().map(file => loadTestCases(file)));

        if (args.filter !== undefined) {
            tests = filterTests(tests, args.filter.split(","));
        }

        const jiraApi = new Jira(args.jiraUsername, args.jiraPassword);

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

        const idKeyMap = new Map<string, string>();

        for (const test of tests) {
            const previousRun = previousRuns.find(run => run.id === test.id);

            const issue = toIssue(
                test,
                args.epic,
                project,
                fixVersion.id,
                fixBuild.id,
                toPriority(previousRun)
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
                idKeyMap.set(test.id, result.key);

                if (previousRun) {
                    await jiraApi.addLinkToIssue(
                        result.key,
                        toIssueLink(previousRun)
                    );
                    logger.info(`   linked to '${previousRun.issue.key}'`);

                    if (args.autoResolve) {
                        if (
                            previousRun.result === "Passed" ||
                            previousRun.result === "Skipped"
                        ) {
                            await jiraApi.resolveIssue(result.key);
                            logger.info(
                                ` '${result.key}' automatically resolved as "Won't Do"`
                            );
                        }
                    }
                }
            }
        }

        for (const test of tests) {
            const currentTest = idKeyMap.get(test.id);
            for (const requireId of test.require) {
                if (args.dryRun) {
                    logger.info(
                        `will link test '${test.id}' to test '${requireId}`
                    );
                    continue;
                }

                if (!idKeyMap.has(requireId)) {
                    throw new Error(
                        ` Can't link '${test.id}' to '${requireId}' because '${requireId}' isn't a valid test case`
                    );
                }

                const blockerTest = idKeyMap.get(requireId);
                await jiraApi.addLinkToIssue(
                    currentTest,
                    toBlockedByLink(blockerTest)
                );
                logger.info(
                    ` '${currentTest}' linked to '${blockerTest}' as blocked by`
                );
            }
        }
    }
};

export { jira };
