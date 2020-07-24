import * as fs from "fs";
import { Argv, CommandModule } from "yargs";
import { filterTests, loadTestCases } from "../lib/test-case";

interface CSVArgs {
    output: string;
    filter: string;
}

const jql = (id: string) =>
    `project = Integreatly AND labels  = test-case  AND summary ~ "${id}" ORDER BY createdDate  DESC`;

const runsLink = (id: string) =>
    `https://issues.redhat.com/issues/?jql=${encodeURI(jql(id))}`;

// tslint:disable:object-literal-sort-keys
const csv: CommandModule<{}, CSVArgs> = {
    command: "csv",
    describe: "export all test cases in a csv file",
    builder: {
        output: {
            describe: "the name of the file where to write the csv table",
            type: "string",
            demand: true
        },
        filter: {
            describe: "filter test to create by tags",
            type: "string"
        }
    },
    handler: async args => {
        let tests = loadTestCases();

        if (args.filter !== undefined) {
            tests = filterTests(tests, args.filter.split(","));
        }

        const rows = [
            [
                "ID",
                "Category",
                "Title",
                "Tags",
                "Components",
                "Targets",
                "Estimate",
                "Require",
                "Automation Jiras",
                "Link",
                "Runs"
            ].join(",")
        ];

        const data = tests.map(t =>
            [
                t.id,
                t.category,
                t.title,
                t.tags.join(" "),
                t.components.join(" "),
                t.targets.join(" "),
                t.estimate,
                t.require.join(" "),
                t.automation.join(" "),
                t.file.link,
                runsLink(t.id)
            ].join(",")
        );

        rows.push(...data);

        fs.writeFileSync(args.output, rows.join("\n"));
    }
};

const expor: CommandModule = {
    command: "export",
    describe: "export the test cases in csv",
    builder: (args: Argv): Argv => {
        return args.command(csv);
    },
    handler: () => {
        // nothing
    }
};

export { expor };
