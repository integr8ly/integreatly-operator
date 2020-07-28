import { Issue, Jira, Resolution } from "./jira";
import { extractId } from "./test-case";

type Result = "Passed" | "Failed" | "Blocked" | "Skipped" | "ToDo";

interface TestRun {
    id: string;
    title: string;
    link: string;
    result: Result;
    issue: Issue;
}

function resolutionToResult(resolution: Resolution): Result {
    if (!resolution) {
        return "ToDo";
    }
    switch (resolution.name) {
        case "Done":
            return "Passed";
        case "Rejected":
            return "Failed";
        case "Deferred":
            return "Blocked";
        case "Won't Do":
            return "Skipped";
        default:
            throw new Error(`'${resolution.name}' is not a vailid resolution`);
    }
}

async function loadTestRuns(jira: Jira, filter: string): Promise<TestRun[]> {
    const issues = await jira.searchIssues(filter);

    return issues.issues.map((i) => {
        const { id } = extractId(i.fields.summary);
        const link = `https://issues.redhat.com/browse/${i.key}`;
        const result = resolutionToResult(i.fields.resolution);

        return {
            id,
            issue: i,
            link,
            result,
            title: i.fields.summary,
        };
    });
}

export { loadTestRuns, TestRun };
