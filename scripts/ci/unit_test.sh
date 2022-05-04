#!/usr/bin/env bash
set -e

COVER_PROFILE=coverage.out

report_coverage_prow() {
    if [[ -z "${COVERALLS_TOKEN_PATH}" ]]; then
        COVERALLS_TOKEN_PATH="/usr/local/integr8ly-ci-secrets/INTLY_OPERATOR_COVERALLS_TOKEN"
    fi

    if [[ -z "${COVERALLS_TOKEN}" ]]; then
        COVERALLS_TOKEN=$(cat ${COVERALLS_TOKEN_PATH})
    fi

    # need to override prow's BUILD_NUMBER to "" so it won't be reported as jobID to avoid 5xx error :D
    BUILD_NUMBER="" PULL_REQUEST_NUMBER=${PULL_NUMBER} goveralls \
           -coverprofile=$COVER_PROFILE \
           -service=prow \
           -repotoken "$COVERALLS_TOKEN"
}

report_coverage_travis() {
    goveralls -coverprofile=$COVER_PROFILE \
           -service=travis-ci \
           -repotoken "$COVERALLS_TOKEN"
}

echo Running tests:
# tests with negated `unittests` build tag will not be ran
go test -tags=unittests -covermode=count -coverprofile=$COVER_PROFILE.tmp ./apis/... ./controllers/... ./pkg/...
< $COVER_PROFILE.tmp grep -v "zz_generated" > $COVER_PROFILE


if [[ -n "${REPORT_COVERAGE}" ]]; then

    go get github.com/mattn/goveralls
    go install -v github.com/mattn/goveralls

    if [[ -n "${PROW_JOB_ID}" ]]; then
        report_coverage_prow || echo "push to coveralls failed"
    fi

    if [[ -n "${TRAVIS_BUILD_NUMBER}" ]]; then
        report_coverage_travis || echo "push to coveralls failed"
    fi

fi
