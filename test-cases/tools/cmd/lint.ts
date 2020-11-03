import * as path from "path";
import { CommandModule } from "yargs";
import {
    AUTOMATED_TAG,
    DESTRUCTIVE_TAG,
    MANUAL_SELECTION_TAG,
    PER_BUILD_TAG,
    PER_RELEASE_TAG,
    STEPS_SECTION,
} from "../lib/constants";
import {
    desiredFileName,
    isAutomated,
    isPerBuild,
    isPerRelease,
    loadRoughTestCases,
    manualSelectionOnly,
    refineTestCase,
    RoughTestCase,
    TestCase,
} from "../lib/test-case";
import { isEmpty } from "../lib/utils";
import { logger } from "../lib/winston";

type Linter<T> = (test: T) => error;

type error = string | null;

const AUTOMATION = /^[A-Z]+-[0-9]+$/;

// Update the README.md too
const PRODUCTS = ["rhmi", "rhoam"];

const CATEGORIES = [
    "alerts",
    "authorization",
    "backup-restore",
    "dashboards",
    "documentation",
    "high-availability",
    "installation",
    "monitoring",
    "multiAZ",
    "performance",
    "products",
    "upgrade",
    "walkthroughs",
    "uninstallation",
];

// Update the README.md too
const COMPONENTS = [
    "monitoring",
    "product-ups",
    "product-codeready",
    "product-apicurito",
    "product-amq",
    "product-3scale",
    "product-sso",
    "product-fuse",
    "product-data-sync",
];

// Update the README.md too
const ENVIRONMENTS = [
    "osd-fresh-install",
    "osd-post-upgrade",
    "osd-private-post-upgrade",
    "rhpds",
    "external",
];

const TARGETS = /^[0-9]+\.[0-9]+\.[0-9]+$/;

// Update the README.md too
const TAGS = [
    PER_BUILD_TAG,
    PER_RELEASE_TAG,
    AUTOMATED_TAG,
    DESTRUCTIVE_TAG,
    MANUAL_SELECTION_TAG,
];

// Update the test-template.md to
const SECTIONS = [STEPS_SECTION, "Description", "Prerequisites"];

function lintFileNames(): Linter<RoughTestCase> {
    return (test: RoughTestCase): error => {
        const desired = desiredFileName(test);

        const { base: current } = path.parse(test.file);

        if (current !== desired) {
            return `${current} should be renamed to ${desired}`;
        }
        return null;
    };
}

function lintDuplicateIDs(): Linter<RoughTestCase> {
    const parsed: { [id: string]: RoughTestCase } = {};

    return (test: RoughTestCase): error => {
        if (test.id in parsed) {
            return `the id: ${test.id} is duplicated in '${
                parsed[test.id].file
            }' and in '${test.file}'`;
        }
        parsed[test.id] = test;
        return null;
    };
}

function lintCategories(): Linter<RoughTestCase> {
    return lintStringField(
        "category",
        includes(CATEGORIES),
        `valid categories are: ${CATEGORIES}`
    );
}

function lintAutomationJiras(): Linter<RoughTestCase> {
    return lintStringArrayField(
        "automation",
        regex(AUTOMATION),
        `the automation ticket must respect the jira format: ${AUTOMATION}`
    );
}

function lintComponents(): Linter<RoughTestCase> {
    return lintStringArrayField(
        "components",
        includes(COMPONENTS),
        `valid components are: ${COMPONENTS}`
    );
}

function lintProducts(): Linter<TestCase> {
    return lintStringField(
        "productName",
        includes(PRODUCTS),
        `valid products are: ${PRODUCTS}`
    );
}

function lintEnvironments(): Linter<TestCase> {
    return lintStringArrayField(
        "environments",
        includes(ENVIRONMENTS),
        `valid environments are: ${ENVIRONMENTS}`
    );
}

function lintTargets(): Linter<TestCase> {
    return lintStringArrayField(
        "targets",
        regex(TARGETS),
        `the target version must respect the this format: ${TARGETS}`
    );
}

function lintTags(): Linter<RoughTestCase> {
    return lintStringArrayField(
        "tags",
        includes(TAGS),
        `valid tags are: ${TAGS}`
    );
}

function lintStringField<T>(
    field: string,
    l: (f: string) => boolean,
    tip: string
): Linter<T> {
    return (test: T): error => {
        if (l(test[field])) {
            return `invalid ${field}: ${test[field]}, ${tip}`;
        }
        return null;
    };
}

function lintStringArrayField<T>(
    field: string,
    l: (f: string) => boolean,
    tip: string
): Linter<T> {
    return (test: T): error => {
        for (const e of test[field]) {
            if (l(e)) {
                return `invalid ${field}: ${e}, ${tip}`;
            }
        }
        return null;
    };
}

function includes(list: string[]): (f: string) => boolean {
    return (f) => !list.includes(f);
}

function regex(reg: RegExp): (f: string) => boolean {
    return (f) => !reg.test(f);
}

function lintMandatoryEnvironment(): Linter<TestCase> {
    return (test: TestCase): error => {
        if (!isAutomated(test) && isEmpty(test.environments)) {
            return `at least one environment must be set for each not automated test cases`;
        }
        return null;
    };
}

function lintOccurrence(): Linter<TestCase> {
    return (test: TestCase): error => {
        if (isAutomated(test) || manualSelectionOnly(test)) {
            return null;
        }

        if (isPerBuild(test) && isPerRelease(test)) {
            return `can not be per-build and per-release at the same time`;
        }

        if (isPerBuild(test) && !isEmpty(test.targets)) {
            return `can not be per-build and have a target version`;
        }

        if (isPerRelease(test) && !isEmpty(test.targets)) {
            return `can not be per-release and have a target version`;
        }

        if (isEmpty(test.targets) && !isPerRelease(test) && !isPerBuild(test)) {
            return `must have a target version or be a per-release or per-build test case`;
        }

        return null;
    };
}

function lintSections(): Linter<RoughTestCase> {
    return (test: RoughTestCase): error => {
        const sections = [];

        const lines = test.content.split("\n");
        for (const line of lines) {
            const match = /^\s*#{2}(?!#)\s+(?<section>.*)\s*$/.exec(line);
            if (match) {
                sections.push(match.groups.section);
            }
        }

        for (const section of sections) {
            if (!SECTIONS.includes(section)) {
                return `invalid section: ${section}, valid sections are: ${SECTIONS}`;
            }
        }

        if (!isAutomated(test) && !sections.includes(STEPS_SECTION)) {
            return `the ${STEPS_SECTION} section is not defined`;
        }

        return null;
    };
}

const roughLinters: { [key: string]: Linter<RoughTestCase> } = {
    "automation-jiras": lintAutomationJiras(),
    categories: lintCategories(),
    components: lintComponents(),
    "duplicate-ids": lintDuplicateIDs(),
    "file-names": lintFileNames(),
    sections: lintSections(),
    tags: lintTags(),
};

const linters: { [key: string]: Linter<TestCase> } = {
    environments: lintEnvironments(),
    "mandatory-environment": lintMandatoryEnvironment(),
    occurrence: lintOccurrence(),
    products: lintProducts(),
    targets: lintTargets(),
};

// tslint:disable:object-literal-sort-keys
const lint: CommandModule<{}, {}> = {
    command: "lint",
    describe: "verify all test cases",
    builder: {},
    handler: () => {
        const roughs = loadRoughTestCases();

        let dirty = false;
        for (const l of Object.keys(roughLinters)) {
            logger.info(`linting: ${l}`);
            for (const rough of roughs) {
                const err = roughLinters[l](rough);
                if (err !== null) {
                    logger.error(`${l}: ${rough.file}: ${err}`);
                    dirty = true;
                }
            }
        }

        for (const l of Object.keys(linters)) {
            logger.info(`linting: ${l}`);
            for (const rough of roughs) {
                for (const test of refineTestCase(rough)) {
                    const err = linters[l](test);
                    if (err !== null) {
                        logger.error(
                            `${l}: ${test.file}: ${test.productName}: ${err}`
                        );
                        dirty = true;
                    }
                }
            }
        }

        if (dirty) {
            logger.error("linting: some checks failed, see errors above");
            process.exit(1);
        }

        logger.info("linting: all checks succeeded");
    },
};

export { lint, PRODUCTS };
