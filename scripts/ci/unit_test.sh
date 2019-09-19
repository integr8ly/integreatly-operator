#!/usr/bin/env bash
set -e

COVER_PROFILE=coverage.out

report_coverage() {

    if [[ -z "${COVERALLS_TOKEN_PATH}" ]]; then
        COVERALLS_TOKEN_PATH="/usr/local/integr8ly-ci-secrets/INTLY_OPERATOR_COVERALLS_TOKEN"
    fi

    if [[ -z "${COVERALLS_TOKEN}" ]]; then
        COVERALLS_TOKEN=$(cat ${COVERALLS_TOKEN_PATH})
    fi

    go get github.com/mattn/goveralls
    go install -v github.com/mattn/goveralls
    # need to override prow's BUILD_NUMBER to "" so it won't be reported as jobID to avoid 5xx error :D
    BUILD_NUMBER="" PULL_REQUEST_NUMBER=${PULL_NUMBER} goveralls \
           -coverprofile=$COVER_PROFILE \
           -service=prow \
           -repotoken $COVERALLS_TOKEN
}

echo Running tests:
go test -v -covermode=count -coverprofile=$COVER_PROFILE ./pkg/...

if [[ -z "${PROW_JOB_ID}" ]]; then
    echo "Not a CI job, skipping coverage reporting!"
else
    report_coverage
fi
