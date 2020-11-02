import * as fs from "fs";
import { Argv, CommandModule } from "yargs";
import {
    filterTests,
    loadTestCases,
    releaseFilter,
    stringToFilter,
} from "../lib/test-case";

interface CSVArgs {
    output?: string;
    environment?: string;
    product?: string;
    target?: string;
    filter?: string[];
}

const jql = (id: string) =>
    `project = Integreatly AND labels  = test-case  AND summary ~ "${id}" ORDER BY createdDate  DESC`;

const runsLink = (id: string) =>
    `https://issues.redhat.com/issues/?jql=${encodeURI(jql(id))}`;

// tslint:disable:object-literal-sort-keys
const csv: CommandModule<{}, CSVArgs> = {
    command: "csv",
    describe: "export all test cases in a csv file or print them to stdout",
    builder: {
        output: {
            describe: "the name of the file where to write the csv table",
            type: "string",
        },
        environment: {
            describe:
                "the environment name used from the release filter, if not set the release filter will not be used",
            type: "string",
        },
        product: {
            demand: true,
            describe:
                "the product name used from the release filter, if not set the release filter will not be used",
            type: "string",
        },
        target: {
            describe:
                "the target version used from the release filter, if not set the release filter will not be used",
            type: "string",
        },
        filter: {
            describe: "filter test to create by most of the fields",
            type: "array",
        },
    },
    handler: async (args) => {
        if (
            (args.environment && !args.target) ||
            (args.target && !args.environment)
        ) {
            throw new Error(
                "if environment is passed also target must be passed and vice versa"
            );
        }

        let tests = loadTestCases(args.product);

        if (args.target || args.environment) {
            tests = releaseFilter(tests, args.environment, args.target);
        }

        if (args.filter !== undefined) {
            tests = filterTests(tests, stringToFilter(args.filter));
        }

        const rows = [
            [
                "ID",
                "Category",
                "Title",
                "Tags",
                "Environments",
                "Components",
                "Targets",
                "Estimate",
                "Automation Jiras",
                "Link",
                "Runs",
            ].join(","),
        ];

        const data = tests.map((t) =>
            [
                t.id,
                t.category,
                t.title,
                t.tags.join(" "),
                t.environments.join(" "),
                t.components.join(" "),
                t.targets.join(" "),
                t.estimate,
                t.automation.join(" "),
                t.url,
                runsLink(t.id),
            ].join(",")
        );

        rows.push(...data);

        if (args.output) {
            fs.writeFileSync(args.output, rows.join("\n"));
        } else {
            rows.forEach((r) => console.log(r));
        }
    },
};

const expor: CommandModule = {
    command: "export",
    describe: "export the test cases in csv",
    builder: (args: Argv): Argv => {
        return args.command(csv);
    },
    handler: () => {
        // nothing
    },
};

export { expor };
