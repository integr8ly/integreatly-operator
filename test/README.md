# Tests

## Structures

* [`common`](./common)

  This is where you should add your tests. They will be included in both the `e2e` and `functional` suites. See the next section about how to add tests here.

* [`e2e`](./e2e)
  This is used to initialise and run tests defined in `common` directory using the operator-sdk. Existing tests there should be refactored to move to the `common` directory.
  
  To run the `e2e` tests, execute:
  
  ```
  make test/e2e
  ```
  
  Because operator-sdk is used, they are useful for developing tests locally. You can run them against a clean OpenShift cluster, and operator-sdk will automatically clean up when the tests are finished.
 
* [`functional`](./functional)

  This is used to initialise and run tests defined in `common` directory using `go test`. You can invoke the tests here against an existing RHMI cluster either using `go test`, or from an IDE.
  
  To run the `functional` tests, make sure you have logged in to the cluster first, then execute:
  ```
  make test/functional
  ```
  
  This is useful if you need to run the tests against an existing RHMI cluster for debugging or verification purposes.
  
  It will also be used to build the test harness image, and run tests from inside an container on a cluster. Note that in order to run the tests, the container needs to have `admin` permission. 
  
  This test harness image will be used as part of our own testing pipelines, as well as as part of the [OSD Addon testing flow](https://github.com/openshift/osde2e/blob/master/docs/Addons.md). 

* [`metadata`](./metadata)  
  
  This directory is required by the OSD Addon testing, more details can be found [here](https://docs.google.com/document/d/1sqpJ0ChJeya3QdsnIOiLDyOqCMF48OaOQkPoyDxjO48/edit#heading=h.1ow8wgpb44i5). 
  
  Normally speaking, you don't need to make any changes here.

* [`scorecard`](./scorecard)  

  This directory contains source code for the KUTTL scorecard test image
  * `entrypoint` - a shell script used for triggering KUTTL tests within the image
  * `main.go` - A helper tool for processing the output from kuttl test and converting it to the scorecard test status format
## Adding New Common Tests

You should aim to add new tests here so that they can be executed in both test suites.

* The test should be created in the `common` directory
* If a new test file is needed, **DO NOT** use `_test` as the suffix of the file name.
* Each test should have the following signature:
    ```
    func TestSomething(t *testing.T, ctx *TestingContext) {
       //Implement the test
    }
    ```
  The [`TestingContext`](./common/types.go) object contains a few clients that you can use in your tests. They are initialised automatically according to the environment that the tests are executed in.
* Make sure new tests are added to the `ALL_TESTS` array that is defined in the [common/tests.go](./common/tests.go) file.
* As the test will be executed in different environments, try not to use functions that are provided by the operator-sdk's testing framework if you can.

## Adding New Tests For A Single Suite

Generally speaking all test cases should be added to the `common` directory. However, if you are sure that a test should be only executed as part of 1 suite, you should:

* Add the test to the test suite's corresponding directory
* If a new test file is created, it should follow Golang's convention for testing. E.g. use `_test` as the suffix of the file name.

## Build Functional Test Image

Run the following command in the root of the `integreatly-operator` directory:

```
make image/functional/build
```

and push it to quay.io:

```
make image/functional/push
```