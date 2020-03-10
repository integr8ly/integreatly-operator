# Test Best Practices

This is a collection of best practices to write high quality tests. The best practices
can be referenced in review comments to speed up the review process.

- [Go Style](#go-style)
- [Test Case Traceability](#test-case-traceability)
- [Don't Sleep](#dont-sleep)
- [Independent](#independent)
- [Secrets](#secrets)
- [Logging](#logging)

## Go Style

We don't want to redefine best practices for go, therefore we should reuse the existing ones
described in the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
and in the[Effective Go](https://golang.org/doc/effective_go.html) documents.

## Test Case Traceability

In order to compute test coverage and test automation progress across multiple test suites all
automated tests must have an ID and must be traced back to the [integreatly-test-cases](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases)
repo following the [How to automate a test case and link it back](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases#how-to-automate-a-test-case-and-link-it-back)
tutorial.

## Don't Sleep

When waiting for a resource to become available or an operation to complete do not use fixed wait time,
instead you should create a loop and keep pooling until the operation doesn't complete.

## Independent

A test must always be independent of other tests, which mean:

- it should not rely on other tests changes so that it would be possible to run it independently

- it should clean up after himself so that it would be possible to run it multiple time

- it should not rely on a static order when querying data because multiple tests may run in parallel

- it should not delete or modify other tests data because multiple tests may run in parallel

- it should not destruct any resource because other tests may rely on them

## Secrets

Do not commit or log any sensitive data or secrets, and always double check for it because it can happen unconsciously

## Logging

Do not be afraid of logging, if the test pass we will ignore the logs but if the test fail logs will help
understand why the test has failed, especially if the tests fail during an unexpected step.

Use the [`t.Log`](https://golang.org/pkg/testing/#B.Log) to print to the console instead of `fmt.Print`.
