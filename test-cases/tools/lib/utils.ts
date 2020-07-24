function extractId(title: string): { id: string; title: string } {
    // A01 - Title
    const match = /^(?<id>[A-Z][0-9]{2})\s-\s(?<title>.*)$/.exec(title);
    if (match) {
        return {
            id: match.groups.id,
            title: match.groups.title
        };
    } else {
        throw new Error(`can not extract the ID from '${title}'`);
    }
}

function isEmpty(list: any[]): boolean {
    return !(list.length > 0);
}

export { extractId, isEmpty };
