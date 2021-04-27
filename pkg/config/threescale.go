package config

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/test/resources"

	threescaleapps "github.com/3scale/3scale-operator/pkg/apis/apps"
	threescalev1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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
			TypeMeta: metav1.TypeMeta{
				Kind:       threescaleapps.APIManagerKind,
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

func (t *ThreeScale) GetOperatorNamespace() string {
	return t.config["OPERATOR_NAMESPACE"]
}

func (t *ThreeScale) SetOperatorNamespace(newNamespace string) {
	t.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (t *ThreeScale) GetLabelSelector() string {
	return "middleware"
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

func (t *ThreeScale) GetReplicasConfig(inst *integreatlyv1alpha1.RHMI) map[string]int64 {
	threeScaleComponents := map[string]int64{
		"systemApp":       3,
		"systemSidekiq":   3,
		"apicastProd":     3,
		"apicastStage":    3,
		"backendListener": 3,
		"backendWorker":   3,
		"backendCron":     1,
		"zyncApp":         3,
		"zyncQue":         3,
	}

	if resources.RunningInProw(inst) {
		setDefaultNumberOfReplicas(1, threeScaleComponents)
	} else if inst.Spec.Type != string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		setDefaultNumberOfReplicas(2, threeScaleComponents)
	}

	if inst.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		switch inst.Status.Quota {
		case "1":
			threeScaleComponents["apicastProd"] = 1
			threeScaleComponents["backendListener"] = 1
			threeScaleComponents["backendWorker"] = 1
		case "5":
			threeScaleComponents["apicastProd"] = 3
			threeScaleComponents["backendListener"] = 3
			threeScaleComponents["backendWorker"] = 3
		case "10":
			threeScaleComponents["apicastProd"] = 3
			threeScaleComponents["backendListener"] = 3
			threeScaleComponents["backendWorker"] = 3
		case "20":
			threeScaleComponents["apicastProd"] = 3
			threeScaleComponents["backendListener"] = 3
			threeScaleComponents["backendWorker"] = 3
		case "50":
			threeScaleComponents["apicastProd"] = 3
			threeScaleComponents["backendWorker"] = 4
			threeScaleComponents["backendListener"] = 5
		}
	}

	return threeScaleComponents
}

func setDefaultNumberOfReplicas(defaultNumberOfReplicas int64, threeScaleComponents map[string]int64) {
	for i := range threeScaleComponents {
		threeScaleComponents[i] = int64(defaultNumberOfReplicas)
	}
}
