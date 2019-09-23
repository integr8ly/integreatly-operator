package controller

import (
	"github.com/aerogear/mobile-security-service-operator/pkg/controller/mobilesecurityservicebackup"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, mobilesecurityservicebackup.Add)
}
