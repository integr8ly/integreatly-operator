import http from 'k6/http';

export default function () {
    const res = http.get('<3scale api stagin url from example curl request here>');
}