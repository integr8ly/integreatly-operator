package config

import (
	"errors"

	webapp "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type SolutionExplorer struct {
	config ProductConfig
}

func NewSolutionExplorer(config ProductConfig) *SolutionExplorer {
	return &SolutionExplorer{config: config}
}
func (s *SolutionExplorer) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&webapp.WebApp{
			TypeMeta: v1.TypeMeta{
				Kind:       "WebApp",
				APIVersion: webapp.SchemeGroupVersion.String(),
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

func (s *SolutionExplorer) Read() ProductConfig {
	return s.config
}

func (s *SolutionExplorer) GetHost() string {
	return s.config["HOST"]
}

func (r *SolutionExplorer) GetLabelSelector() string {
	return "middleware"
}

func (r *SolutionExplorer) GetTemplateList() []string {
	template_list := []string{
		"kube_state_metrics_solution_explorer_alerts.yaml",
	}
	return template_list
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
