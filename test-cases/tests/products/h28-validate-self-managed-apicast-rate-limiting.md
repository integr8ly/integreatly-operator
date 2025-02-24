---
products:
  - name: rhoam
    environments:
      - osd-post-upgrade
    targets:
      - 1.8.0
      - 1.11.0
      - 1.14.0
      - 1.20.0
      - 1.23.0
      - 1.26.0
      - 1.29.0
      - 1.30.0
      - 1.33.0
      - 1.35.0
      - 1.38.0
      - 1.39.0
      - 1.42.0
estimate: 30m
tags:
  - destructive
---

# H28 - Validate Self Managed Apicast rate limiting

## Description

This test case should prove that the rate limiting works as expected if Self Managed Apicast is deployed and used in the RHOAM cluster.

## Prerequisites

Self Managed Apicast to be deployed on the RHOAM cluster. See [H24 test case](./h24-verify-selfmanaged-apicast-and-custom-policy.md) on how to do it (in short, run H24 test case locally with SKIP_CLEANUP=true).

In order to use production (managed) APIcast (optional for this test case) use the default "API" Product and promote to both Stage and Production. In order to use production self-managed APIcast you need to update the route in 3scale Admin Portal (H24 test only uses staging APIcast), promote it to Production and create the route in self-managed APIcast namespace (simplest way is to get the yaml for existing staging APIcast's route and change the '.spec.host' there).

Workload web app not deployed on the cluster

## Steps

**1. Communication between Managed Apicast and Backend Listener is not rate limited.**

- Create a request via Managed APIcast and make sure it is counted just once.
- Create a request via Self Managed APIcast and make sure it is counted just once.

Note: to create a request you can use `curl` as follows. It prints `200` and datetime if everything goes ok:

```
curl -s -o /dev/null -w "%{http_code}" "<your-apicast-url>/?user_key=<your-user-key>" && echo -n " - " && date && echo ""
```

Note: Apicasts might not call backend-listeners for each request so in case of Self Managed Apicast slightly more requests might be allowed than expected.

Note: to check the rate limit counter use Rate Limiting Grafana Dashboard or following promQL:

```
sum(increase(authorized_calls[5m])) + sum(increase(limited_calls[5m]))
```

    > Note that limited_calls returns an empty result until there is at least one rate limeted request

Other approach might be to check backend-listeners logs. For each request there should be similar log entry to:

```
10.11.78.11 - - [04/Jun/2021 15:02:49 UTC] "GET /transactions/authrep.xml?service_token=ea41be9d8af467ae2eabf8b3bbd69f1a5d5955676731a70270852b1acdfd2c19&service_id=4&usage%5Bworkload_app_api_metric.3%5D=1&usage%5Bhits%5D=1&user_key=bb5de14fb96734dfaccd5b6ef6722181&log%5Bcode%5D=200 HTTP/1.1" 200 - 0.00170681 0 0 0 15 35448 31593 - "rejection_reason_header=1&limit_headers=1&no_body=1"
```

**2. Verify that both Self Managed and Managed Apicast are returning 429 - Too Many Requests when the limit of requests is reached.**

- Set the limit to 10 requests per minute.
- Do 5 requests via Self Managed Apicast and 5 requests via Managed apicast
- Do one more request via both SM and Managed Apicasts to see that 429 is received for both

Note: Changing requests per minute needs to be done via ‘ratelimit-config’ ConfigMap in marin3r namespace, but rhoam operator needs to be scaled to 0 (zero) beforehand otherwise it reconciles the limit back. Also, all the marin3r pods need to be restarted for the change to be applied.

**3. Make sure there is not any (3scale) internal traffic to Backend Listeners rate-limited**

- Check the current value of ratelimit counter
- Run automated tests, exclude the tests that produce traffic to Backend Listeners
- Create Product/Backend in 3scale Admin portal
- Wait for some time
- Check that the ratelimit counter value stays the same, IE there was not any internal traffic rate-limited
