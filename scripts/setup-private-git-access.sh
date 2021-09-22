#!/usr/bin/env bash

set -e

if [[ -z "$GIT_USERNAME" ]]; then
  echo "GIT_USERNAME must be set in order to configure private repo access"
  exit 1
fi

if [[ -z "$GIT_ACCESS_TOKEN" ]]; then
  echo "GIT_TOKEN must be set in order to configure private repo access"
  exit 1
fi

echo "setting up configuration for pulling private go repos"
go env GOPRIVATE=github.com/bf2fc6cc711aee1a0c2a/observability-operator
git config --global url."https://${GIT_USERNAME}:${GIT_ACCESS_TOKEN}@github.com".insteadOf https://github.com
