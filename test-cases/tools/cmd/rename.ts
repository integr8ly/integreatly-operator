import * as fs from "fs";
import * as path from "path";
import { CommandModule } from "yargs";
import { desiredFileName, loadRoughTestCases } from "../lib/test-case";
import { logger } from "../lib/winston";

// tslint:disable:object-literal-sort-keys
const rename: CommandModule<{}, {}> = {
    command: "rename",
    describe: "rename all test cases files according to the titles",
    builder: {},
    handler: () => {
        const tests = loadRoughTestCases();

        tests.forEach((test) => {
            const desired = desiredFileName(test);

            const { base: current, dir } = path.parse(test.file);

            if (current !== desired) {
                fs.renameSync(test.file, path.join(dir, desired));
                logger.info(`${current} renamed to ${desired}`);
            }
        });
    },
};

export { rename };
