import * as path from "path";
import { CommandModule, string } from "yargs";
import { extractTitle, loadTestCases, TestCase } from "../lib/test-case";
import { desiredFileName, loadTestFiles } from "../lib/test-file";
import { flat } from "../lib/utils";
import { logger } from "../lib/winston";

function lintFileNames() {
    const files = loadTestFiles();

    let dirty = false;
    files.forEach(file => {
        const { title } = extractTitle(file.content);
        const desired = desiredFileName(title);

        const { base: current, dir } = path.parse(file.file);

        if (current !== desired) {
            dirty = true;
            logger.warn(`${current} should be renamed to ${desired}`);
        }
    });

    if (dirty) {
        logger.error(
            "some files are not named correctly, use `./tools.sh rename` to fix them all"
        );
        process.exit(1);
    }
}

function lintIDs() {
    const tests = flat(loadTestFiles().map(file => loadTestCases(file)));

    const parsed: { [id: string]: TestCase } = {};

    let dirty = false;
    tests.forEach(test => {
        if (test.id in parsed) {
            dirty = true;
            logger.warn(
                `the ${test.id} is duplicated in '${parsed[test.id].file.file}' and in '${test.file.file}'`
            );
        } else {
            parsed[test.id] = test;
        }
    });

    if (dirty) {
        logger.error(
            "some IDs are duplicated, you need to select a new ID for the new test cases"
        );
        process.exit(1);
    }
}

// tslint:disable:object-literal-sort-keys
const lint: CommandModule<{}, {}> = {
    command: "lint",
    describe: "verify all test cases",
    builder: {},
    handler: () => {
        logger.info("checking test cases IDs");
        lintIDs();

        logger.info("checking test cases file names");
        lintFileNames();

        logger.info("all test cases are correct");
    }
};

export { lint };
