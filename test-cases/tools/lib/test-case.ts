import * as fs from "fs";
import * as matter from "gray-matter";
import * as path from "path";
import { extractId } from "./utils";
import { boolean } from "yargs";
import { AUTOMATED_TAG } from "./constants";

const TEST_DIR = "./tests";
const TEST_FILTER = /^.*\.md$/;
const REPO_URL =
    "https://github.com/integr8ly/integreatly-operator/tree/master/test-cases";

interface Metadata {
    automation: string[];
    components: string[];
    environments: string[];
    estimate: string;
    require: string[];
    tags: string[];
    targets: string[];
}

interface TestCase {
    id: string;
    category: string;
    title: string;
    content: string;
    environments: string[];
    estimate: number;
    tags: string[];
    targets: string[];
    components: string[];
    automation: string[];
    require: string[];
    file: string;
    url: string;
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

/**
 * Recursive search for all files in dir that matches the filter.
 */
function walk(dir: string, filter: RegExp): string[] {
    const results: string[] = [];

    for (const file of fs.readdirSync(dir)) {
        const full = path.join(dir, file);

        const stats = fs.statSync(full);

        if (stats.isDirectory()) {
            results.push(...walk(full, filter));
        } else if (filter.test(file)) {
            results.push(full);
        }
    }

    return results;
}

function loadTestCases(testDirectory?: string): TestCase[] {
    return walk(testDirectory || TEST_DIR, TEST_FILTER).map(loadTestCase);
}

function loadTestCase(file: string): TestCase {
    const m = matter.read(file);
    const data = m.data as Metadata;

    const te = extractTitle(m.content);
    let title = te.title;
    const content = te.content;

    const ie = extractId(title);
    const id = ie.id;
    title = ie.title;

    const category = extractCategory(file);

    return {
        automation: data.automation || [],
        category,
        components: data.components || [],
        content,
        environments: data.environments || [],
        estimate: data.estimate ? convertEstimation(data.estimate) : null,
        file,
        id,
        require: data.require || [],
        tags: data.tags || [],
        targets: data.targets || [],
        title,
        url: `${REPO_URL}/${file}`
    };
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

function desiredFileName(test: TestCase): string {
    let name = `${test.id} - ${test.title}`;

    name = name.toLowerCase();
    name = name.replace(/[^a-z0-9\s]/g, "");
    name = name.replace(/\s+/g, "-");
    name = name.substr(0, 64);
    name = name.replace(/-$/, "");

    return `${name}.md`;
}

function isAutomated(test: TestCase): boolean {
    return test.tags.includes(AUTOMATED_TAG);
}

export { loadTestCases, TestCase, filterTests, desiredFileName, isAutomated };
