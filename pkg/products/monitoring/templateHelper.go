package monitoring

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
)

type Parameters struct {
	Params map[string]string
}

type TemplateHelper struct {
	Parameters   Parameters
	TemplatePath string
}

// Creates a new templates helper and populates the values for all
// templates properties. Some of them (like the hostname) are set
// by the user in the custom resource
func NewTemplateHelper(extraParams map[string]string) *TemplateHelper {
	param := Parameters{
		Params: extraParams,
	}

	templatePath := GetTemplatePath()

	return &TemplateHelper{
		Parameters:   param,
		TemplatePath: templatePath,
	}
}

// Validate if the "templates/monitoring" directory exists in the
// filesystem & if yes, returns the same as a string
func GetTemplatePath() string {

	templatePath := "./templates/monitoring"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		templatePath = "../../../templates/monitoring"
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			templatePath = "/usr/local/bin/templates/monitoring"
			if _, err := os.Stat(templatePath); os.IsNotExist(err) {
				panic("cannot find templates")
			}
		}
	}
	return templatePath
}

// Takes a list of strings, wraps each string in double quotes and joins them
// Used for building yaml arrays
func joinQuote(values []monitoring.BlackboxtargetData) string {
	var result []string
	for _, s := range values {
		result = append(result, fmt.Sprintf("\"%v@%v@%v\"", s.Module, s.Service, s.Url))
	}
	return strings.Join(result, ", ")
}

// load a templates from a given resource name. The templates must be located
// under ./templates and the filename must be <resource-name>.yaml
func (h *TemplateHelper) LoadTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s", h.TemplatePath, name)
	tpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := template.New("application-monitoring")
	parser.Funcs(template.FuncMap{
		"JoinQuote": joinQuote,
	})

	parsed, err := parser.Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	err = parsed.Execute(&buffer, h.Parameters)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (h *TemplateHelper) CreateResource(template string) (runtime.Object, error) {
	tpl, err := h.LoadTemplate(template)
	if err != nil {
		return nil, err
	}

	resource := unstructured.Unstructured{}
	err = yaml.Unmarshal(tpl, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
