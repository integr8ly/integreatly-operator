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

func (s *SolutionExplorer) GetNamespace() string {
	return s.config["NAMESPACE"]
}

func (s *SolutionExplorer) SetNamespace(newNamespace string) {
	s.config["NAMESPACE"] = newNamespace
}

func (s *SolutionExplorer) Read() ProductConfig {
	return s.config
}

func (s *SolutionExplorer) GetHost() string {
	return s.config["HOST"]
}

func (s *SolutionExplorer) SetHost(newHost string) {
	s.config["HOST"] = newHost
}

func (s *SolutionExplorer) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductSolutionExplorer
}

func (s *SolutionExplorer) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionSolutionExplorer
}

func (s *SolutionExplorer) Validate() error {
	if s.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if s.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if s.GetProductVersion() == "" {
		return errors.New("version is not defined")
	}
	return nil
}
