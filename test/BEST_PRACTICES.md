# Test Best Practices

This is a collection of best practices to write high quality tests. These best practices should be referenced in the comments that you provide when you perform code reviews of tests.

- [Coding Style](#coding-style)
- [Test Case Traceability](#test-case-traceability)
- [Don't Sleep](#dont-sleep)
- [Independent](#independent)
- [Secrets](#secrets)
- [Logging](#logging)
- [Don't fail immediately](#dont-fail-immediately)

## Coding Style

The code style should be consistent with the style used in the integreatly-operator codebase [TODO: link to golang practices used by engineering]

Also, in case of doubt, the Golang standard best practices can be used:

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html) documents.

## Test Case Traceability

In order to compute test coverage and test automation progress across multiple test suites all automated tests must have an ID and must be traced back to the [integreatly-test-cases](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases) repo following the [How to automate a test case and link it back](https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases#how-to-automate-a-test-case-and-link-it-back) tutorial.

## Don't Sleep

When waiting for a resource to become available or an operation to complete do not use fixed wait times and/or sleep statements, instead you should create a loop to check if the resource is available or the operation has completed

## Independent

A test must always be independent of other tests, which means:

- it should not rely on the successful execution of other tests so the test can be run independently

- it should clean up after itself so it is possible to run the test concurrently or multiple times

- it should not delete or modify resources used by other tests so tests may run in parallel

## Secrets

Ensure secret or sensitive data is not included in commits or log output.

## Logging

Do not be afraid of logging, if the test pass we will ignore the logs but if the test fail logs will help understand why the test has failed, especially if the tests fail during an unexpected step.

Use the [`t.Log`](https://golang.org/pkg/testing/#B.Log) to print to the console instead of `fmt.Print`.

## Don't fail immediately

When verifying multiple resources or performing steps that do not depend on each other, try to not fail on the first error mark the test as failed and log the error using the [`t.Error`](https://golang.org/pkg/testing/#B.Error) method and proceed with the test execution to test as much as possible.
