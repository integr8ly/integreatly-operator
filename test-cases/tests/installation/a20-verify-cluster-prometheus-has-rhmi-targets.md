---
environments:
  - osd-post-upgrade
estimate: 15m
targets:
  - 2.3.0
  - 2.6.0
---

# A20 - Verify cluster Prometheus has RHMI targets

## Steps

1. Get targets from RHMI prometheus API (in `redhat-rhmi-middleware-monitoring-operator` namespace)
   - e.g. <https://prometheus-route-redhat-rhmi-middleware-monitoring-operator.apps...s1.devshift.org/api/v1/targets>
   - save it to `targetsA.json` file
2. Get targets from cluster prometheus API (in `openshift-monitoring` namespace)
   - e.g. <https://prometheus-k8s-openshift-monitoring.apps...s1.devshift.org/api/v1/targets>
   - save it to `targetsB.json` file
3. Use this nodejs script to compare targets:

   ```js
   const targetsA = require("./targetsA.json");
   const targetsB = require("./targetsB.json");

   const toIgnore = [
     "openshift-monitoring-federation",
     "3scale-apicast-pods",
     "blackbox",
     "redhat-rhmi-rhsso/keycloak-pod-monitor",
     "redhat-rhmi-user-sso/keycloak-pod-monitor",
   ];

   const summarize = (prev, curr) => {
     if (!prev[curr.labels.job]) {
       prev[curr.labels.job] = {
         num: 0,
         upNum: 0,
       };
     }
     prev[curr.labels.job].num++;
     if (curr.health === "up") {
       prev[curr.labels.job].upNum++;
     }
     return prev;
   };

   const resultA = targetsA.data.activeTargets.reduce(summarize, {});
   const resultB = targetsB.data.activeTargets.reduce(summarize, {});

   let ok = true;

   for (const key in resultA) {
     if (!toIgnore.includes(key)) {
       if (!resultB[key]) {
         console.log(`B does not contain '${key}' target`);
         ok = false;
         continue;
       }
       if (resultA[key].num > resultB[key].num) {
         console.log(
           `B has different number of '${key}' targets: A-${resultA[key].num} vs B-${resultB[key].num}`
         );
         ok = false;
         continue;
       }
       if (resultA[key].upNum > resultB[key].upNum) {
         console.log(`B has different number of '${key}' targets that are UP`);
         ok = false;
         continue;
       }
     }
   }

   if (ok) {
     console.log("Looks good");
   } else {
     console.log("Test failed");
   }
   ```
