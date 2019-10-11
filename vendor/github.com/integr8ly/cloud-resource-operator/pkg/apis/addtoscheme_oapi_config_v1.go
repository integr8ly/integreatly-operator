package apis

import (
	"github.com/openshift/api/config/v1"
)

func init() {
	AddToSchemes = append(AddToSchemes, v1.Install)
}
