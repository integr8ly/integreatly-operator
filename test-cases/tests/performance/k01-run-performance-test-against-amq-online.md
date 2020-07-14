---
estimate: 4h
tags:
  - amq
---

# K01 - Run performance test against AMQ Online

Note: this test should only be performed at a time it will not affect other ongoing testing, or on a separate cluster.

## Steps

1. Follow the manual steps in [AMQ Online load testing](https://github.com/integr8ly/middleware-load-testing/tree/master/amq-online#manual-setup-and-execution)
2. Capture the results and compare them against [the previous run](https://docs.google.com/spreadsheets/d/1AotZFy7ugcAdxKm0ToNZN4tlEVpWKJyeZ9pBhcPZSbw/edit#gid=1064657125), take note of the `About Tests` section on the sheets when comparing results. As slight alterations to the tests can greatly influence results
3. If running maestro with the postgres db exporter, the max latency is not recorded. Depending on number of tests expected to be run more information can be extracted from the maestro web interface
4. Expecting similar results
5. It is important to run the AMQ Perf Testing tool on a separate instance to the test cluster as maestro is quite resource heavy. The config used previously can be found [here](https://github.com/integr8ly/middleware-load-testing/pull/7)
