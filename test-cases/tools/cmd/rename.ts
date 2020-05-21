import * as fs from "fs";
import * as path from "path";
import { CommandModule } from "yargs";
import { extractTitle } from "../lib/test-case";
import { loadTestFiles } from "../lib/test-file";
import { logger } from "../lib/winston";

function desiredFileName(title: string): string {
    let name = title;

    name = name.toLowerCase();
    name = name.replace(/[^a-z0-9\s]/g, "");
    name = name.replace(/\s+/g, "-");
    name = name.substr(0, 64);
    name = name.replace(/-$/, "");

    return `${name}.md`;
}

interface Args {
    write: boolean;
}

// tslint:disable:object-literal-sort-keys
const rename: CommandModule<{}, Args> = {
    command: "rename",
    describe: "check and list all tests that should be renamed",
    builder: {
        write: {
            alias: "w",
            describe: "physically rename all tests",
            boolean: true,
            default: false
        }
    },
    handler: args => {
        const files = loadTestFiles();

        let dirty = false;
        files.forEach(file => {
            const { title } = extractTitle(file.content);
            const desired = desiredFileName(title);

            const { base: current, dir } = path.parse(file.file);

            if (current !== desired) {
                if (args.write) {
                    fs.renameSync(file.file, path.join(dir, desired));
                    logger.info(`${current} renamed to ${desired}`);
                } else {
                    dirty = true;
                    logger.warn(`${current} should be renamed to ${desired}`);
                }
            }
        });

        if (!args.write && dirty) {
            logger.error(
                "some files are not named correctly, use --write to rename all of them"
            );
            process.exit(1);
        }
    }
};

export { rename };
