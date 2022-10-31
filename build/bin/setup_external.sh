#!/bin/sh

oc login ${OPENSHIFT_HOST} -u kubeadmin -p ${OPENSHIFT_PASSWORD}
SUITE_COMMAND="/integreatly-operator-test-harness.test -test.v -ginkgo.v -ginkgo.progress -ginkgo.no-color"
if [[ ! -z "${RegExpFilter}" ]]; then
    SUITE_COMMAND="${SUITE_COMMAND} -ginkgo.focus ${RegExpFilter}"
fi
$SUITE_COMMAND | tee test-run-results/log.txt
