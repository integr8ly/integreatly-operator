package controller

import (
	"github.com/aerogear/mobile-developer-console-operator/pkg/controller/mobiledeveloperconsole"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, mobiledeveloperconsole.Add)
}
