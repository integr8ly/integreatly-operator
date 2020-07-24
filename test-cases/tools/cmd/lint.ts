import * as path from "path";
import { CommandModule } from "yargs";
import { desiredFileName, loadTestCases, TestCase } from "../lib/test-case";
import { logger } from "../lib/winston";

type Linter = (test: TestCase) => error;

type error = string | null;

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
            return `the ID: ${test.id} is duplicated in '${parsed[test.id].file}' and in '${test.file}'`;
        }
        parsed[test.id] = test;
        return null;
    };
}

const linters: { [key: string]: Linter } = {
    "duplicate-ids": lintDuplicateIDs(),
    "file-names": lintFileNames()
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
                    logger.error(`${l}: ${err}`);
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
