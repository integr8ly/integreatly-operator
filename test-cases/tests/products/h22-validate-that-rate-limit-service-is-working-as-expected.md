---
automation:
  - MGDAPI-1261
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
      - osd-fresh-install
      - osd-private-post-upgrade
    targets:
      - 1.0.0
      - 1.3.0
      - 1.6.0
      - 1.7.0
estimate: 1h
---

# H22 - Validate that Rate Limit service is working as expected

## Description

This test case should prove that the rate limiting Redis counter correctly increases with every request made

## Steps

1. Open Openshift Console in your browser
2. Copy `oc login` command and login to your cluster in the terminal
3. Start the automated test run
   ```sh
   ./test/scripts/products/h22-validate-that-rate-limit-service-is-working-as-expected/test.sh | tee test-output.txt
   ```
4. Wait for the test to finish. Ensure that it finished successfully, the output
   should look like this:
   ```
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
   ℹ️  Redis host: ratelimit-service-redis-rhoam.redhat-rhoam-operator.svc.cluster.local
     🔑  Found access token: **********
     ✔️  Created Account ID: 9
     ✔️  Created Backend ID: 8
     ✔️  Created Metric ID: 24
     ✔️  Mapping rule created
     ✔️  Created Service ID: 8
     ✔️  Backend usage created
     ✔️  Created Application Plan ID: 20
     ✔️  User key: **********
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ✔️  Proxy deployed
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ✔️  Promoted proxy. Endpoint: https://h22-test-6-api-3scale-apicast-production.apps.sfrancog.41x3.s1.devshift.org
   ️🔌  Created API. Endpoint https://h22-test-6-api-3scale-apicast-production.apps.sfrancog.41x3.s1.devshift.org?user_key=****************
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
     ☁️  Waiting for throwaway Redis container to complete. Current phase: in progress...
   ️ℹ️  Throw away Redis Pod ready
     Redis pod name: throw-away-redis-pod-57f8ddf94f-klwff
   Previous count: 0 | Number of requests: 12 | Current count: 12 | [PASS] 🎉
   Previous count: 0 | Number of requests: 5 | Current count: 5 | [PASS] 🎉
   Previous count: 0 | Number of requests: 14 | Current count: 14 | [PASS] 🎉
   Previous count: 0 | Number of requests: 11 | Current count: 11 | [PASS] 🎉
   Previous count: 0 | Number of requests: 11 | Current count: 11 | [PASS] 🎉
   ```
