import * as JiraApi from "jira-client";

const JIRA_HOST = "issues.redhat.com";
const JIRA_PROTOCOL = "https";

type IssueTypeName = "Epic" | string;

interface Resolution {
    name: string;
}

interface IssueLink {
    type: {
        name: string;
    };
    outwardIssue?: { key: string };
    inwardIssue?: { key: string };
}

interface Issue {
    key?: string;
    fields: {
        versions: { id: string; name?: string }[];
        fixVersions: { id: string; name?: string }[];
        customfield_12312442?: { id: string; value?: string }; // fixBuild
        customfield_12311140: string;
        customfield_12313240: string; // team
        customfield_12310940: string[] | number; // sprint: type 'string[]' when listing, type 'number' when creating a new jira
        description: string;
        issuetype: { name: IssueTypeName };
        labels: string[];
        project: { key: string };
        priority?: { name: string };
        issuelinks?: IssueLink[];
        summary: string;
        assignee: null | {};
        resolution?: Resolution;
        components: null | [{}];
        security: { name?: string };
    };
    [name: string]: any;
}

interface Issues {
    issues: Issue[];
}

function isEpic(issue: Issue) {
    return issue.fields.issuetype.name === "Epic";
}

function assertEpic(issue: Issue) {
    if (!isEpic(issue)) {
        throw new Error(`${issue.key} is not an Epic`);
    }
}

class Jira {
    public client;

    public constructor(token: string) {
        // @ts-ignore
        this.client = new JiraApi({
            host: JIRA_HOST,
            protocol: JIRA_PROTOCOL,
            bearer: token,
        });
    }

    public findIssue(key: string): Promise<Issue> {
        return this.client.findIssue(key) as Promise<Issue>;
    }

    public searchIssues(jql: string): Promise<Issues> {
        return this.client.searchJira(jql) as Promise<Issues>;
    }

    public addNewIssue(issue: Issue): Promise<Issue> {
        return this.client.addNewIssue(issue) as Promise<Issue>;
    }

    public issueLink(link: IssueLink): Promise<unknown> {
        return this.client.issueLink(link) as Promise<unknown>;
    }

    public resolveIssue(key: string): Promise<unknown> {
        return this.client.transitionIssue(key, {
            fields: { resolution: { name: "Won't Do" } },
            transition: { id: "51" },
        }) as Promise<unknown>;
    }
}

export { Jira, isEpic, assertEpic, Issue, Issues, Resolution };
