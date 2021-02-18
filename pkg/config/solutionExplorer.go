package config

import (
	"errors"

	solutionExplorerv1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/tutorial-web-app-operator/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

type SolutionExplorer struct {
	config ProductConfig
}

func NewSolutionExplorer(config ProductConfig) *SolutionExplorer {
	return &SolutionExplorer{config: config}
}
func (s *SolutionExplorer) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&solutionExplorerv1alpha1.WebApp{
			TypeMeta: metav1.TypeMeta{
				Kind:       "WebApp",
				APIVersion: solutionExplorerv1alpha1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (s *SolutionExplorer) GetNamespace() string {
	return s.config["NAMESPACE"]
}

func (s *SolutionExplorer) SetNamespace(newNamespace string) {
	s.config["NAMESPACE"] = newNamespace
}

func (s *SolutionExplorer) GetOperatorNamespace() string {
	return s.config["OPERATOR_NAMESPACE"]
}

func (s *SolutionExplorer) SetOperatorNamespace(newNamespace string) {
	s.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (s *SolutionExplorer) Read() ProductConfig {
	return s.config
}

func (s *SolutionExplorer) GetHost() string {
	return s.config["HOST"]
}

func (s *SolutionExplorer) GetLabelSelector() string {
	return "middleware"
}

func (s *SolutionExplorer) SetHost(newHost string) {
	s.config["HOST"] = newHost
}

func (s *SolutionExplorer) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductSolutionExplorer
}

func (s *SolutionExplorer) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(s.config["VERSION"])
}

func (s *SolutionExplorer) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionSolutionExplorer
}

func (s *SolutionExplorer) SetProductVersion(newVersion string) {
	s.config["VERSION"] = newVersion
}

func (s *SolutionExplorer) Validate() error {
	if s.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if s.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	return nil
}
