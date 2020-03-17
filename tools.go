// +build tools

package tools

import (
	// Used by "make pkg/apis/integreatly/v1alpha1/zz_generated.openapi.go"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)
