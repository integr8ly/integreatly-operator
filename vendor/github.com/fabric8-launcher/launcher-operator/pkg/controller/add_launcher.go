package controller

import (
	"fabric8-launcher/launcher-operator/pkg/controller/launcher"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, launcher.Add)
}
