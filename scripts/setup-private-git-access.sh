#!/usr/bin/env bash

# This script is used by prow in order to setup the required configuration to pull the private observability operator
# You can use it locally to generate the required config also as long as you have the two required ENV VARs exported
# and they contain a token from your user who should have access to the observability-operator repo

set -e

if [[ -z "$GIT_USERNAME" ]]; then
  echo "GIT_USERNAME must be set in order to configure private repo access"
  echo "if you see this message in a prow job then the job does not have access to the secrets required"
  exit 1
fi

if [[ -z "$GIT_ACCESS_TOKEN" ]]; then
  echo "$GIT_ACCESS_TOKEN must be set in order to configure private repo access"
  echo "if you see this message in a prow job then the job does not have access to the secrets required"
  exit 1
fi

echo "setting up configuration for pulling private go repos"
go env GOPRIVATE=github.com/bf2fc6cc711aee1a0c2a/observability-operator
git config --global url."https://${GIT_USERNAME}:${GIT_ACCESS_TOKEN}@github.com".insteadOf https://github.com
