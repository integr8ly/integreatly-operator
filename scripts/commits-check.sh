#!/usr/bin/env bash
#
# This script retrieves the list of commits difference from master
# and ensures that each commit starts with a MGDAPI- numbered ticket
# See https://issues.redhat.com/browse/MGDAPI-3097
# PULL_BASE_SHA and PULL_PULL_SHA are set by prow

PULL_BASE_SHA="${PULL_BASE_SHA:-$(git merge-base master HEAD)}"
PULL_PULL_SHA="${PULL_PULL_SHA:-HEAD}"

commits=$(git rev-list --no-merges $PULL_BASE_SHA..$PULL_PULL_SHA)
invalidCommits=()

if [ -z "$commits" ]; then
    echo "No commits to scan"
    exit 0
fi

for sha in $commits; do
    msg=$(git log --format=%B -n 1 $sha)
    if [[ ! $msg =~ ^(MGDAPI-[0-9]+)[[:blank:]].*$ ]]; then
        invalidCommits+=( "Commit $sha:\n$msg\n" )
    fi
done

if [ ${#invalidCommits[@]} -gt 0 ]; then
    printf "Not all commits start with a \"MGDAPI-\" ticket\n\n"
    for i in "${invalidCommits[@]}"; do
        printf "$i"
    done
    exit 1
fi

echo "All commits valid"
