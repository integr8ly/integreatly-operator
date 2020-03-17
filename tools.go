// +build tools

package tools

import (
	// Used by make if correct version not on PATH
	_ "github.com/operator-framework/operator-sdk/cmd/operator-sdk"

	// Used by "make pkg/apis/integreatly/v1alpha1/zz_generated.openapi.go"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)
