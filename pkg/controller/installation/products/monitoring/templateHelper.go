package monitoring

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	monitoring_v1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
)

const (
	Placeholder = "XXX"
)

type Parameters struct {
	Placeholder   string
	Namespace     string
	MonitoringKey string
	ExtraParams   map[string]string
}

type TemplateHelper struct {
	Parameters   Parameters
	TemplatePath string
}

// Creates a new templates helper and populates the values for all
// templates properties. Some of them (like the hostname) are set
// by the user in the custom resource
func newTemplateHelper(cr *v1alpha1.Installation, extraParams map[string]string, config *config.Monitoring) *TemplateHelper {
	param := Parameters{
		Placeholder:   Placeholder,
		Namespace:     config.GetNamespace(),
		MonitoringKey: config.GetLabelSelector(),
		ExtraParams:   extraParams,
	}

	templatePath, exists := os.LookupEnv("TEMPLATE_PATH")
	if !exists {
		templatePath = "./templates/monitoring"
	}

	monitoringKey, exists := os.LookupEnv("MONITORING_KEY")
	if exists {
		param.MonitoringKey = monitoringKey
	}

	return &TemplateHelper{
		Parameters:   param,
		TemplatePath: templatePath,
	}
}

// Takes a list of strings, wraps each string in double quotes and joins them
// Used for building yaml arrays
func joinQuote(values []monitoring_v1alpha1.BlackboxtargetData) string {
	var result []string
	for _, s := range values {
		result = append(result, fmt.Sprintf("\"%v@%v@%v\"", s.Module, s.Service, s.Url))
	}
	return strings.Join(result, ", ")
}

// load a templates from a given resource name. The templates must be located
// under ./templates and the filename must be <resource-name>.yaml
func (h *TemplateHelper) loadTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.yaml", h.TemplatePath, name)
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
