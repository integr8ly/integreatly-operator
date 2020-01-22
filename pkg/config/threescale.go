package config

import (
	"errors"

	threescalev1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type ThreeScale struct {
	config ProductConfig
}

func NewThreeScale(config ProductConfig) *ThreeScale {
	return &ThreeScale{config: config}
}
func (t *ThreeScale) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&threescalev1alpha1.APIManager{
			TypeMeta: v1.TypeMeta{
				Kind:       "APIManager",
				APIVersion: threescalev1alpha1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (t *ThreeScale) GetHost() string {
	return t.config["HOST"]
}

func (t *ThreeScale) SetHost(newHost string) {
	t.config["HOST"] = newHost
}

func (t *ThreeScale) GetBlackboxTargetPathForAdminUI() string {
	return t.config["BLACKBOX_TARGET_PATH_ADMIN_UI"]
}

func (t *ThreeScale) SetBlackboxTargetPathForAdminUI(newBlackboxTargetPath string) {
	t.config["BLACKBOX_TARGET_PATH_ADMIN_UI"] = newBlackboxTargetPath
}

func (t *ThreeScale) GetNamespace() string {
	return t.config["NAMESPACE"]
}

func (r *ThreeScale) GetLabelSelector() string {
	return "middleware"
}

func (r *ThreeScale) GetTemplateList() []string {
	templateList := []string{
		"kube_state_metrics_3scale_alerts.yaml",
	}
	return templateList
}

func (t *ThreeScale) SetNamespace(newNamespace string) {
	t.config["NAMESPACE"] = newNamespace
}

func (t *ThreeScale) Read() ProductConfig {
	return t.config
}

func (t *ThreeScale) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.Product3Scale
}

func (t *ThreeScale) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(t.config["VERSION"])
}

func (t *ThreeScale) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersion(t.config["OPERATOR"])
}

func (t *ThreeScale) SetOperatorVersion(operator string) {
	t.config["OPERATOR"] = operator
}

func (t *ThreeScale) SetProductVersion(newVersion string) {
	t.config["VERSION"] = newVersion
}

func (t *ThreeScale) Validate() error {
	if t.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if t.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if t.GetHost() == "" {
		return errors.New("config host is not defined")
	}
	return nil
}
