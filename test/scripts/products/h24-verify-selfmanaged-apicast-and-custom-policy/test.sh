#!/bin/bash

#  Prerequisites:
# 		oc logged as cluster admin (kubeadmin)
#  Recommended, before running the script in interactive mode (- interactive-mode=true ), to have following:
# 		Customer Token, if testing-idp and customer-admin users are already available.
# 		Open 3scale Admin Portal, to be ready to put updates. Details will be notified by script.
#  If script executed in batch mode (-interactive-mode=false) - no input required, just to be logged as admin.

go run h24-verify-selfmanaged-apicast-and-custom-policy.go \
-apicast-operator-version="0.5.0" \
-apicast-image-stream-tag="3scale2.11.0" \
-apicast-namespace="selfmanaged-apicast" \
-use-customer-admin-user=true \
-interactive-mode=false \
--namespace-prefix redhat-rhoam- \


