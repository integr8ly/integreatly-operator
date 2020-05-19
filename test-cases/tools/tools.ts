import * as yargs from "yargs";
import { expor, jira, polarion, rename } from "./cmd";

// tslint:disable-next-line:no-unused-expression
yargs
    .command(jira)
    .command(rename)
    .command(polarion)
    .command(expor).argv;
