import * as winston from "winston";

export const logger: winston.Logger = winston.createLogger({
    format: winston.format.json(),
    level: "info",
    transports: [
        new winston.transports.Console({
            format: winston.format.simple(),
        }),
    ],
});
