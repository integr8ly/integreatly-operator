package controller

import (
	"github.com/RHsyseng/operator-utils/pkg/resource/detector"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, []string, *detector.Detector) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, products []string, detector *detector.Detector) error {

	for _, f := range AddToManagerFuncs {
		if err := f(m, products, detector); err != nil {
			return err
		}
	}
	return nil
}
