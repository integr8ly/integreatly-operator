import http from 'k6/http';

export let options = {
  scenarios: {
    constant_request_rate: {
      executor: 'constant-arrival-rate',
      rate: Number.parseInt(`${__ENV.RPM}`),
      timeUnit: '1m',
      duration: '10m',
      preAllocatedVUs: 10,
      maxVUs: 20
    },
  },
};

export default function () {
    const res = http.get('<3scale api stagin url from example curl request here>');
}