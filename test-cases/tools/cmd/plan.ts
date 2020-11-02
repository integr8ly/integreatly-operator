import { Argv, CommandModule } from "yargs";
import {
    isAutomated,
    isPerBuild,
    isPerRelease,
    loadTestCases,
    manualSelectionOnly,
    updateTargets,
} from "../lib/test-case";
import { logger } from "../lib/winston";

// A test case will be bring in release each 3 releases
const CYCLE = 3;

interface Version {
    major: number;
    minor: number;
    patch: number;
}

function stringToVersion(version: string): Version {
    const match = /^(?<ma>\d+)\.(?<mi>\d+)\.(?<pa>\d+)$/.exec(version);
    if (!match) {
        throw new Error(`invalid version: ${version}`);
    }

    return {
        major: parseInt(match.groups.ma, 10),
        minor: parseInt(match.groups.mi, 10),
        patch: parseInt(match.groups.pa, 10),
    };
}

function versionToString(version: Version): string {
    return `${version.major}.${version.minor}.${version.patch}`;
}

interface ReleaseArgs {
    product: string;
    target: string;
    dryRun: boolean;
}

// tslint:disable:object-literal-sort-keys
const release: CommandModule<{}, ReleaseArgs> = {
    command: "release",
    describe:
        "automatically assign the target version to the test cases which latest version is older than 3 minor releases",
    builder: {
        product: {
            demand: true,
            describe: "the product to set the version to",
            type: "string",
        },
        target: {
            demand: true,
            describe: "the version to check against and set to the test cases",
            type: "string",
        },
        "dry-run": {
            type: "boolean",
        },
    },
    handler: (args) => {
        const version = stringToVersion(args.target);
        if (version.patch !== 0) {
            logger.error(
                "the plan release cmd can be used only for minor release"
            );
            process.exit(1);
        }

        const tests = loadTestCases(args.product);

        for (const test of tests) {
            // skip all test cases that are automated, per-release, per-build or marked as manual-selection
            if (
                isAutomated(test) ||
                isPerRelease(test) ||
                isPerBuild(test) ||
                manualSelectionOnly(test)
            ) {
                continue;
            }

            let latest: number = -CYCLE;
            for (const t of test.targets) {
                let v: Version;
                try {
                    v = stringToVersion(t);
                } catch (e) {
                    logger.error(
                        `failed to parse version in test case ${test.file} with error: ${e}`
                    );
                    continue;
                }

                if (v.major !== version.major) {
                    // ignore target version for different major versions
                    continue;
                }

                if (v.minor > latest) {
                    latest = v.minor;
                }
            }

            // if the latest target version on the test case is n version older then the
            // passed version add the passed version to the test case
            if (latest + CYCLE <= version.minor) {
                const v = versionToString(version);
                const targets = test.targets;
                targets.push(v);

                logger.info(
                    `add target ${v} to ${test.id} - ${test.title} | ${test.file}`
                );
                if (!args.dryRun) {
                    updateTargets(test, args.product, targets);
                }
            }
        }
    },
};

interface ForArgs {
    product: string;
    target: string;
    component: string;
    dryRun: boolean;
}

// tslint:disable:object-literal-sort-keys
const forcmd: CommandModule<{}, ForArgs> = {
    command: "for",
    describe:
        "automatically assign the target version to the test cases with the passed component",
    builder: {
        product: {
            demand: true,
            describe: "all the test cases with this product will be updated",
            type: "string",
        },
        target: {
            demand: true,
            describe: "the version to set to the test cases",
            type: "string",
        },
        component: {
            demand: true,
            describe: "all the test cases with this component will be updated",
            type: "string",
        },
        "dry-run": {
            type: "boolean",
        },
    },
    handler: (args) => {
        const tests = loadTestCases(args.product);

        for (const test of tests) {
            // skip all test cases that are automated, per-release, per-build or marked as manual-selection
            if (
                isAutomated(test) ||
                isPerRelease(test) ||
                isPerBuild(test) ||
                manualSelectionOnly(test)
            ) {
                continue;
            }

            if (
                test.components.includes(args.component) &&
                !test.targets.includes(args.target)
            ) {
                const targets = test.targets;
                targets.push(args.target);

                logger.info(
                    `add target ${args.target} to ${test.id} - ${test.title} | ${test.file}`
                );
                if (!args.dryRun) {
                    updateTargets(test, args.product, targets);
                }
            }
        }
    },
};

const plan: CommandModule = {
    command: "plan",
    describe: "automatically assign the target version to the test cases",
    builder: (args: Argv): Argv => {
        return args.command(release).command(forcmd);
    },
    handler: () => {
        // nothing
    },
};

export { plan };
