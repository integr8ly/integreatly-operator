import * as fs from "fs";
import * as matter from "gray-matter";
import * as path from "path";
import {
    AUTOMATED_TAG,
    DESTRUCTIVE_TAG,
    MANUAL_SELECTION_TAG,
    PER_BUILD_TAG,
    PER_RELEASE_TAG,
} from "./constants";
import { flat, walk } from "./utils";

const TEST_DIR = "./tests";
const TEST_FILTER = /^.*\.md$/;
const REPO_URL =
    "https://github.com/integr8ly/integreatly-operator/tree/master/test-cases";

interface Metadata {
    automation: string[];
    components: string[];
    estimate: string;
    products: Product[];
    tags: string[];
}

interface Filter {
    include: string[];
    exclude: string[];
}

interface Filters {
    id?: Filter;
    category?: Filter;
    environments?: Filter;
    products?: Filter;
    tags?: Filter;
    targets?: Filter;
    components?: Filter;
}

interface RoughTestCase {
    id: string;
    category: string;
    title: string;
    content: string;
    estimate: number;
    products: Product[];
    tags: string[];
    components: string[];
    automation: string[];
    file: string;
    url: string;
    matter: matter.GrayMatterFile<string>;
}

interface TestCase extends RoughTestCase {
    productName: string;
    environments: string[];
    targets: string[];
}

interface Product {
    name: string;
    targets: string[];
    environments: string[];
}

function extractTitle(content: string): { title: string; content: string } {
    const lines = content.split("\n");
    while (lines) {
        const line = lines.shift();
        const match = /^\s*#{1}(?!#)\s+(?<title>.*)\s*$/.exec(line);
        if (match) {
            return {
                content: lines.join("\n"),
                title: match.groups.title,
            };
        }
    }

    throw Error("title not found");
}

function extractId(title: string): { id: string; title: string } {
    // A01a - Title
    const match = /^(?<id>[A-Z][0-9]{2}[AB]?)\s-\s(?<title>.*)$/.exec(title);
    if (match) {
        return {
            id: match.groups.id,
            title: match.groups.title,
        };
    } else {
        throw new Error(`can not extract the ID from '${title}'`);
    }
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

function loadTestCases(
    productName: string,
    testDirectory?: string
): TestCase[] {
    return flat(loadRoughTestCases(testDirectory).map(refineTestCase)).filter(
        (t) => t.productName === productName
    );
}

/**
 * If the test case defines multiple product then return a test case for each product
 */
function refineTestCase(test: RoughTestCase): TestCase[] {
    const result: TestCase[] = [];
    for (const product of test.products) {
        result.push({
            productName: product.name || "",
            environments: product.environments || [],
            targets: product.targets || [],
            ...test,
        });
    }
    return result;
}

function loadRoughTestCases(testDirectory?: string): RoughTestCase[] {
    return walk(testDirectory || TEST_DIR, TEST_FILTER).map((f) =>
        loadRoughTestCase(f)
    );
}

function loadRoughTestCase(file: string): RoughTestCase {
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
        estimate: data.estimate ? convertEstimation(data.estimate) : null,
        file,
        id,
        matter: m,
        products: data.products || [],
        tags: data.tags || [],
        title,
        url: `${REPO_URL}/${file}`,
    };
}

function updateTargets(
    test: TestCase,
    productName: string,
    targets: string[]
): void {
    for (const i in test.matter.data.products) {
        if (test.matter.data.products[i].name === productName) {
            test.matter.data.products[i].targets = targets;
            break;
        }
    }
    const out = test.matter.stringify("");
    fs.writeFileSync(test.file, out);
}

function stringToFilter(filters: string[]): Filters {
    const r: Filters = {};

    for (const filter of filters) {
        const [n, ff] = filter.split("=");

        r[n] = { include: [], exclude: [] };
        for (const f of ff.split(",")) {
            if (f.startsWith("^")) {
                r[n].exclude.push(f.slice(1));
            } else {
                r[n].include.push(f);
            }
        }
    }

    return r;
}

function filterTests(tests: TestCase[], filters: Filters): TestCase[] {
    return tests.filter((test) => {
        for (const f of Object.keys(filters)) {
            const filter: Filter = filters[f];
            const field: string | string[] = test[f];

            if (filter === undefined) {
                continue;
            }

            if (filter.include !== undefined) {
                for (const include of filter.include) {
                    // exclude tests that don't contain the include condition
                    if (Array.isArray(field)) {
                        if (!field.includes(include)) {
                            return false;
                        }
                    } else {
                        if (field !== include) {
                            return false;
                        }
                    }
                }
            }

            if (filter.exclude !== undefined) {
                for (const exclude of filter.exclude) {
                    // exclude tests that contain the exclude condition
                    if (Array.isArray(field)) {
                        if (field.includes(exclude)) {
                            return false;
                        }
                    } else {
                        if (field === exclude) {
                            return false;
                        }
                    }
                }
            }
        }

        return true;
    });
}

/**
 * The release filter is the filter applied to the test cases to generate the testing Epic
 */
function releaseFilter(
    tests: TestCase[],
    environment: string,
    target: string
): TestCase[] {
    return tests.filter((test) => {
        if (isAutomated(test)) {
            // exclude automated tests
            return false;
        }

        if (!test.environments.includes(environment)) {
            // exclude all tests that are not part of the targeted env
            return false;
        }

        if (isPerBuild(test) || isPerRelease(test)) {
            // include all test that are marked as per-build or per-release
            return true;
        }

        if (test.targets.includes(target)) {
            // include all test with the matched target version
            return true;
        }

        // exclude anything else
        return false;
    });
}

function desiredFileName(test: RoughTestCase): string {
    let name = `${test.id} - ${test.title}`;

    name = name.toLowerCase();
    name = name.replace(/[^a-z0-9\s]/g, "");
    name = name.replace(/\s+/g, "-");
    name = name.substr(0, 64);
    name = name.replace(/-$/, "");

    return `${name}.md`;
}

function isAutomated(test: RoughTestCase): boolean {
    return test.tags.includes(AUTOMATED_TAG);
}

function isPerBuild(test: TestCase): boolean {
    return test.tags.includes(PER_BUILD_TAG);
}

function isPerRelease(test: TestCase): boolean {
    return test.tags.includes(PER_RELEASE_TAG);
}

function isDestructive(test: TestCase): boolean {
    return test.tags.includes(DESTRUCTIVE_TAG);
}

function manualSelectionOnly(test: TestCase): boolean {
    return test.tags.includes(MANUAL_SELECTION_TAG);
}

export {
    loadTestCases,
    loadRoughTestCases,
    refineTestCase,
    RoughTestCase,
    TestCase,
    filterTests,
    desiredFileName,
    isAutomated,
    isPerBuild,
    isPerRelease,
    isDestructive,
    manualSelectionOnly,
    extractId,
    stringToFilter,
    releaseFilter,
    updateTargets,
};
