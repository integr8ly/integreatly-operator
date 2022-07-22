package grafana

import (
	"context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	defaultDeploymentNamespace = "mock-namespace"
	defaultDeploymentName      = "mock-name"
)

func TestReconciler_scaleDeployment(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultDeploymentName,
			Namespace: defaultDeploymentNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env: []corev1.EnvVar{},
						},
					},
				},
			},
		},
	}

	client := fakeclient.NewFakeClientWithScheme(basicScheme, deployment)
	reconciler := getBasicReconciler()

	tests := []struct {
		testName            string
		deploymentName      string
		deploymentNamespace string
		scaleValue          int32
		want                integreatlyv1alpha1.StatusPhase
		wantErr             string
	}{
		{
			testName:            "sucessfully scale deployment to 1",
			deploymentName:      defaultDeploymentName,
			deploymentNamespace: defaultDeploymentNamespace,
			scaleValue:          1,
			want:                integreatlyv1alpha1.PhaseCompleted,
			wantErr:             "",
		},
		{
			testName:            "failed when trying to scale nonexistent deployment",
			deploymentName:      "bad-deployment",
			deploymentNamespace: defaultDeploymentNamespace,
			scaleValue:          1,
			want:                integreatlyv1alpha1.PhaseFailed,
			wantErr:             "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			phase, err := reconciler.scaleDeployment(context.TODO(), client, tt.deploymentName, tt.deploymentNamespace, tt.scaleValue)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("scaleDeployment() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if phase != tt.want {
				t.Errorf("scaleDeployment() returned %v for the phase but wanted %v", phase, tt.want)
				return
			}
		})
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := appsv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := operatorsv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

func basicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
		},
	}
}

func getBasicReconciler() *Reconciler {
	return &Reconciler{
		installation: basicInstallation(),
		log:          l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductGrafana}),
		Config:       &config.Grafana{},
	}
}
