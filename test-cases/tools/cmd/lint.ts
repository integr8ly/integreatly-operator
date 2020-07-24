import * as path from "path";
import { CommandModule } from "yargs";
import {
    AUTOMATED_TAG,
    PER_BUILD_TAG,
    PER_RELEASE_TAG
} from "../lib/constants";
import {
    desiredFileName,
    isAutomated,
    loadTestCases,
    TestCase,
    isPerRelease,
    isPerBuild
} from "../lib/test-case";
import { logger } from "../lib/winston";
import { isEmpty } from "../lib/utils";

type Linter = (test: TestCase) => error;

type error = string | null;

const AUTOMATION = /^[A-Z]+-[0-9]+$/;

const CATEGORIES = [
    "alerts",
    "authorization",
    "backup-restore",
    "dashboards",
    "documentation",
    "high-availability",
    "installation",
    "monitoring",
    "performance",
    "products",
    "upgrade",
    "walkthroughs"
];

const COMPONENTS = [
    "monitoring",
    "product-ups",
    "product-codeready",
    "product-apicurito",
    "product-amq",
    "product-3scale",
    "product-sso",
    "product-fuse",
    "product-data-sync"
];

const ENVIRONMENTS = [
    "osd-fresh-install",
    "osd-post-upgrade",
    "osd-private-post-upgrade",
    "rhpds",
    "external"
];

const TARGETS = /^[0-9]+\.[0-9]+\.[0-9]+$/;

const TAGS = [PER_BUILD_TAG, PER_RELEASE_TAG, AUTOMATED_TAG];

function lintFileNames(): Linter {
    return (test: TestCase): error => {
        const desired = desiredFileName(test);

        const { base: current } = path.parse(test.file);

        if (current !== desired) {
            return `${current} should be renamed to ${desired}`;
        }
        return null;
    };
}

function lintDuplicateIDs(): Linter {
    const parsed: { [id: string]: TestCase } = {};

    return (test: TestCase): error => {
        if (test.id in parsed) {
            return `the id: ${test.id} is duplicated in '${parsed[test.id].file}' and in '${test.file}'`;
        }
        parsed[test.id] = test;
        return null;
    };
}

function lintCategories(): Linter {
    return lintStringField(
        "category",
        includes(CATEGORIES),
        `valid categories are ${CATEGORIES}`
    );
}

function lintAutomationJiras(): Linter {
    return lintStringArrayField(
        "automation",
        regex(AUTOMATION),
        `the automation ticket must respect the jira format ${AUTOMATION}`
    );
}

function lintComponents(): Linter {
    return lintStringArrayField(
        "components",
        includes(COMPONENTS),
        `valid components are ${COMPONENTS}`
    );
}

function lintEnvironments(): Linter {
    return lintStringArrayField(
        "environments",
        includes(ENVIRONMENTS),
        `valid environments are ${ENVIRONMENTS}`
    );
}

function lintTargets(): Linter {
    return lintStringArrayField(
        "targets",
        regex(TARGETS),
        `the target version must respect the this format: ${TARGETS}`
    );
}

function lintTags(): Linter {
    return lintStringArrayField(
        "tags",
        includes(TAGS),
        `valid tags are ${TAGS}`
    );
}

function lintStringField(
    field: string,
    l: (f: string) => boolean,
    tip: string
): Linter {
    return (test: TestCase): error => {
        if (l(test[field])) {
            return `the ${field}: ${test[field]} is not valid, ${tip}`;
        }
        return null;
    };
}

function lintStringArrayField(
    field: string,
    l: (f: string) => boolean,
    tip: string
): Linter {
    return (test: TestCase): error => {
        for (const e of test[field]) {
            if (l(e)) {
                return `the ${field}: ${e} is not valid, ${tip}`;
            }
        }
        return null;
    };
}

function includes(list: string[]): (f: string) => boolean {
    return f => !list.includes(f);
}

function regex(reg: RegExp): (f: string) => boolean {
    return f => !reg.test(f);
}

function lintMandatoryEnvironment(): Linter {
    return (test: TestCase): error => {
        if (!isAutomated(test) && isEmpty(test.environments)) {
            return `at least one environment must be set for each not automated test cases`;
        }
        return null;
    };
}

function lintOccurrence(): Linter {
    return (test: TestCase): error => {
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

const linters: { [key: string]: Linter } = {
    "automation-jiras": lintAutomationJiras(),
    categories: lintCategories(),
    components: lintComponents(),
    "duplicate-ids": lintDuplicateIDs(),
    environments: lintEnvironments(),
    "file-names": lintFileNames(),
    "mandatory-environment": lintMandatoryEnvironment(),
    occurrence: lintOccurrence(),
    tags: lintTags(),
    targets: lintTargets()
};

// tslint:disable:object-literal-sort-keys
const lint: CommandModule<{}, {}> = {
    command: "lint",
    describe: "verify all test cases",
    builder: {},
    handler: () => {
        const tests = loadTestCases();

        let dirty = false;
        for (const l of Object.keys(linters)) {
            logger.info(`linting: ${l}`);

            for (const test of tests) {
                const err = linters[l](test);
                if (err !== null) {
                    logger.error(`${l}: ${test.file}: ${err}`);
                    dirty = true;
                }
            }
        }

        if (dirty) {
            logger.error("linting: some checks failed, see errors above");
            process.exit(1);
        }

        logger.info("linting: all checks succeeded");
    }
};

export { lint };
