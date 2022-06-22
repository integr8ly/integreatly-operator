#!/bin/bash

set -e

cd "$(dirname "$0")"

if [[ ! -d "./node_modules" ]]; then
    npm install
fi

exec npx --shell sh ts-node --transpile-only ./tools/tools.ts "$@"
