#!/bin/bash

#  Prerequisites:
# 		oc logged as cluster admin (kubeadmin)
#  Recommended, before running current version of script, to have following:
# 		Password for customer, if testing-idp is not yet available.
# 		Customer Token, if testing-idp and customer-admin users are already available.
# 		Open 3scale Admin Portal, to be ready to put updates. Details will be notified by script.


go run h24-verify-selfmanaged-apicast-and-custom-policy.go \
-apicast-operator-version="0.5.0" \
-apicast-image-stream-tag="3scale2.11.0" \
-apicast-namespace="selfmanaged-apicast" \
-create-testing-idp=false \
-use-customer-admin-user=true \
-promote-manually=true \
--namespace-prefix redhat-rhoam- \


