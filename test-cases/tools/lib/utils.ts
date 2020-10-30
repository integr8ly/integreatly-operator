import * as fs from "fs";
import * as path from "path";

function isEmpty(list: any[]): boolean {
    return !(list.length > 0);
}

/**
 * Recursive search for all files in dir that matches the filter.
 */
function walk(dir: string, filter: RegExp): string[] {
    const results: string[] = [];

    for (const file of fs.readdirSync(dir)) {
        const full = path.join(dir, file);

        const stats = fs.statSync(full);

        if (stats.isDirectory()) {
            results.push(...walk(full, filter));
        } else if (filter.test(file)) {
            results.push(full);
        }
    }

    return results;
}

function flat<T>(array: T[][]): T[] {
    let result = [];
    array.forEach((item) => {
        result = result.concat(item);
    });
    return result;
}

export { isEmpty, walk, flat };
