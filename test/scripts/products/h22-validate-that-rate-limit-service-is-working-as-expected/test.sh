#!/bin/bash

go run ./scripts/products/h22-validate-that-rate-limit-service-is-working-as-expected/h22-validate-that-rate-limit-service-is-working-as-expected.go -- \
    --namespace-prefix redhat-rhoam- \
    --iterations 10