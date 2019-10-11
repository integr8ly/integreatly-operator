package apis

import (
	"github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
)

func init() {
	AddToSchemes = append(AddToSchemes, v1.AddToScheme)
}
