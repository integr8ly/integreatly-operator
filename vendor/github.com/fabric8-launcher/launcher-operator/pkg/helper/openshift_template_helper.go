package helper

import (
	"fmt"
	"io/ioutil"

	"github.com/integr8ly/operator-sdk-openshift-utils/pkg/api/template"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
)

// OpenshiftTemplateHelper ... Helper for resource manipulation
type OpenshiftTemplateHelper struct {
}

// NewOpenshiftTemplateHelper ... create a new ResourceHelper
func NewOpenshiftTemplateHelper() *OpenshiftTemplateHelper {
	return &OpenshiftTemplateHelper{}
}

// Load ... Load a resource from a template name
func (r *OpenshiftTemplateHelper) Load(config *rest.Config, templatePath string, name string) (*template.Tmpl, error) {
	var err error
	path := fmt.Sprintf("%s/%s.yaml", templatePath, name)
	templateData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	jsonData, err := yaml.ToJSON(templateData)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(config, jsonData)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

// Process ... Process a template with params
func (r *OpenshiftTemplateHelper) Process(tmpl *template.Tmpl, namespace string, params map[string]string) (*template.Tmpl, error) {
	var err error

	err = tmpl.Process(params, namespace)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
