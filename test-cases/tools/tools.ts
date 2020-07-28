import * as yargs from "yargs";
import { expor, jira, polarion, rename } from "./cmd";
import { lint } from "./cmd/lint";

// tslint:disable-next-line:no-unused-expression
yargs
    .command(lint)
    .command(jira)
    .command(rename)
    .command(polarion)
    .command(expor).argv;
