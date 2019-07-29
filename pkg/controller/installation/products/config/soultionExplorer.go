package config

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type SolutionExplorer struct {
	config ProductConfig
}

func NewSolutionExplorer(config ProductConfig) *SolutionExplorer {
	return &SolutionExplorer{config: config}
}

func (a *SolutionExplorer) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *SolutionExplorer) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *SolutionExplorer) Read() ProductConfig {
	return a.config
}

func (a *SolutionExplorer) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductSolutionExplorer
}

func (a *SolutionExplorer) Validate() error {
	if a.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if a.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	return nil
}
