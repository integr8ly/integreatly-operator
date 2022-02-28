---
components:
  - product-amq
products:
  - name: rhmi
    environments:
      - osd-post-upgrade
    targets:
      - 2.3.0
      - 2.8.0
estimate: 4h
tags:
  - destructive
---

# K01 - Run performance test against AMQ Online

Note: This test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

1. Copy your cluster token and cluster api url to trigger the pipeline
2. Trigger the [pipeline](https://master-jenkins-csb-intly.apps.ocp-c1.prod.psi.redhat.com/job/Integreatly/job/rhmi-amq-scale-test/) for running the test

> Note: The pipeline would fail with an assertion error in the last stage,as the test is expected to run in cluster with customer like configuration to reach the assert value, but the test results can be compared with the results from previous run for release testing to see if they closely matches.

3. Results are archived in the Jenkins job
4. Download the results folder and see plot-data/messaging-performance.csv

5. Update and compare the results in this [sheet](https://docs.google.com/spreadsheets/d/1Iyjp2JhWdxOJ9KLNibT3oYdvK0BharIAHKCmGvKgk_I/edit#gid=0) to see if the results closely matches with the results from the previous run

> Note: Report a bug if there is a performance degradation from the previous version
