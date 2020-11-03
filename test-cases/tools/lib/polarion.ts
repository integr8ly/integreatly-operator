import axios from "axios";
import * as FormData from "form-data";
import * as https from "https";
import * as xml2js from "xml2js";
import { logger } from "./winston";

const POLARION_IMPORTER_URL =
    "https://polarion.engineering.redhat.com/polarion/import";

/**
 *  Upload the xml data to polarion and wait for the job to complete
 *
 * @param {"testcase" | "xunit"} type
 * @param {string} document
 * @param {string} username
 * @param {string} password
 */
async function uploadToPolarion(
    type: "xunit" | "testcase",
    document: object,
    username: string,
    password: string,
    dumpOnly: boolean = false
) {
    // create the xml
    const builder = new xml2js.Builder();
    const xml = builder.buildObject(document);

    if (dumpOnly) {
        logger.info(xml);
        return;
    }

    // create axios api
    const importer = axios.create({
        auth: { username, password },
        baseURL: POLARION_IMPORTER_URL,
        httpsAgent: new https.Agent({
            rejectUnauthorized: false,
        }),
    });

    logger.info("uploading testcases to polarion");
    const form = new FormData();
    form.append("file", xml, {
        contentType: "text/xml",
        filename: "file.xml",
    });

    const result = await importer.post(type, form, {
        headers: form.getHeaders(),
    });

    // console.log(result.data);
    // return;

    const jobId = result.data.files["file.xml"]["job-ids"][0];
    logger.info(`Job started with id ${jobId}`);
    logger.info(
        `logs: https://polarion.engineering.redhat.com/polarion/import/${type}-log?jobId=${jobId}`
    );
    logger.info("Wait for job to complete");

    wait: while (true) {
        // wait 2s
        await new Promise((r) => setTimeout(r, 2000));

        const status = await importer.get(`${type}-queue`, {
            headers: { Accept: "application/json" },
            params: { jobIds: jobId },
        });

        const job = status.data.jobs[0];
        switch (job.status) {
            case "READY":
            case "RUNNING":
                logger.info(`Job is ${job.status}`);
                continue wait;
            case "SUCCESS":
                logger.info("Job completed successfully");
                break wait;
            default:
                throw new Error(`unknown job status ${job.status}`);
        }
    }
}

// Extract the 3 characters long Test ID
// to keep the backward compatibility
// with existing tests in Polarion
function extractPolarionTestId(id: string): string {
    const found = /[A-Z][0-9]{2}/.exec(id);
    if (found) {
        return found[0];
    } else {
        throw new Error(`cannot extract the polarion test ID from '${id}'`);
    }
}

export { uploadToPolarion, extractPolarionTestId };
