import * as clone from "clone";
import * as merge from "deepmerge";
import * as fs from "fs";
import * as handlebars from "handlebars";
import * as path from "path";
import { Metadata, TestFile } from "./test-file";
import { extractId } from "./utils";

interface TestCase {
    id: string;
    category: string;
    title: string;
    content: string;
    estimate: number;
    tags: string[];
    targets: string[];
    components: string[];
    automationJiras: string[];
    require: string[];
    file: TestFile;
}

/**
 * Handlebars functions
 */
handlebars.registerHelper("lowercase", str => str.toLowerCase());

function expandVariants(
    test: TestFile
): Array<{ data: Metadata; content: string }> {
    const result = [];

    if (test.data.variants !== undefined) {
        const template = handlebars.compile(test.content);
        for (const variant of test.data.variants) {
            // clone metadata
            let data = clone(test.data);

            // remove redundant variants
            delete data.variants;

            // merge data
            data = merge(data, variant);

            // render content
            const content = template(data.vars);

            // remove vars
            delete data.vars;

            result.push({
                content,
                data
            });
        }
    } else {
        result.push({
            content: test.content,
            data: test.data
        });
    }

    return result;
}

function extractTitle(content: string): { title: string; content: string } {
    const lines = content.split("\n");
    while (lines) {
        const line = lines.shift();
        const match = /^\s*#{1}(?!#)\s+(?<title>.*)\s*$/.exec(line);
        if (match) {
            return {
                content: lines.join("\n"),
                title: match.groups.title
            };
        }
    }

    throw Error("title not found");
}

/**
 * Convert estimations in format 1h 2h 30m to a float number where 1 = 1h
 */
function convertEstimation(estimate: string): number {
    const p = /^(\d+)([mh])$/.exec(estimate);
    if (p == null) {
        throw new Error(
            `the estimation '${estimate}' is not in the valid format`
        );
    }

    const [_, amount, unit] = p;
    switch (unit) {
        case "m":
            return parseInt(amount, 10) / 60;
        case "h":
            return parseInt(amount, 10);
        default:
            throw new Error(
                `unexpected unit '${unit}' for estimation '${estimate}'`
            );
    }
}

function extractCategory(file: string): string {
    return path.basename(path.dirname(file));
}

function expandImports(content: string, file: string): string {
    const { dir } = path.parse(file);
    const expanded = [];
    for (const line of content.split("\n")) {
        // import files when matching @ [Some text](./relative/file.md)
        // in the content
        const match = /^@\s*\[.*\]\((?<file>.*)\)\s*$/.exec(line);
        if (match) {
            const fileToImport = path.join(dir, match.groups.file);
            const contentToImport = fs.readFileSync(fileToImport);
            expanded.push(contentToImport);
        } else {
            expanded.push(line);
        }
    }

    return expanded.join("\n");
}

function loadTestCases(file: TestFile): TestCase[] {
    return expandVariants(file).map(({ data, content }) => {
        const titleExtract = extractTitle(content);
        let title = titleExtract.title;
        content = titleExtract.content;

        const idExtract = extractId(title);
        const id = idExtract.id;
        title = idExtract.title;

        const category = extractCategory(file.file);

        content = expandImports(content, file.file);

        const tags = data.tags || [];

        if (data.targets === undefined) {
            tags.push("per-release");
        }

        return {
            category,
            content,
            estimate: data.estimate ? convertEstimation(data.estimate) : null,
            file,
            id,
            require: data.require || [],
            tags: tags,
            components: data.components || [],
            targets: data.targets || [],
            automationJiras: data.automation_jiras || [],
            title
        };
    });
}

function filterTests(tests: TestCase[], filters: string[]): TestCase[] {
    return tests.filter(test => {
        for (let filter of filters) {
            if (filter.startsWith("^")) {
                filter = filter.slice(1);
                if (test.tags.includes(filter)) {
                    // tests with this tag is not included
                    return false;
                }
            } else {
                if (!test.tags.includes(filter)) {
                    // tests without this tag is not included
                    return false;
                }
            }
        }

        return true;
    });
}

export { loadTestCases, TestCase, extractTitle, filterTests };
