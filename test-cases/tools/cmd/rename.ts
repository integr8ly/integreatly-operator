import * as fs from "fs";
import * as path from "path";
import { CommandModule } from "yargs";
import { extractTitle } from "../lib/test-case";
import { desiredFileName, loadTestFiles } from "../lib/test-file";
import { logger } from "../lib/winston";

// tslint:disable:object-literal-sort-keys
const rename: CommandModule<{}, {}> = {
    command: "rename",
    describe: "rename all test cases files according to the titles",
    builder: {},
    handler: () => {
        const files = loadTestFiles();

        files.forEach(file => {
            const { title } = extractTitle(file.content);
            const desired = desiredFileName(title);

            const { base: current, dir } = path.parse(file.file);

            if (current !== desired) {
                fs.renameSync(file.file, path.join(dir, desired));
                logger.info(`${current} renamed to ${desired}`);
            }
        });
    }
};

export { rename };
