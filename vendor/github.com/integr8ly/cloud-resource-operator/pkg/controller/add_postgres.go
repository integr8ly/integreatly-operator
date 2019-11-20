package controller

import (
	"github.com/integr8ly/cloud-resource-operator/pkg/controller/postgres"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, postgres.Add)
}
