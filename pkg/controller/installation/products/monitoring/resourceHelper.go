package monitoring

import (
	"github.com/ghodss/yaml"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourceHelper struct {
	templateHelper *TemplateHelper
	cr             *v1alpha1.Installation
}

func NewResourceHelper(cr *v1alpha1.Installation, th *TemplateHelper) *ResourceHelper {
	return &ResourceHelper{
		templateHelper: th,
		cr:             cr,
	}
}

func (r *ResourceHelper) CreateResource(template string) (runtime.Object, error) {
	tpl, err := r.templateHelper.loadTemplate(template)
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
