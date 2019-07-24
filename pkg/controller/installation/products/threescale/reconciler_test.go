package threescale

import (
	"context"
	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = aerogearv1.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestThreeScale(t *testing.T) {

	integreatlyOperatorNamespace := "integreatly-operator-namespace"
	clusterPreReqObjects, appsv1PreReqObjects := GetClusterPreReqObjects(integreatlyOperatorNamespace, defaultInstallationNamespace)
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error getting pre req objects for %s: %v", packageName, err)
	}
	configManager, fakeSigsClient, fakeAppsV1Client, fakeOauthClient, fakeThreeScaleClient, mpm, err := getClients(clusterPreReqObjects, scheme, appsv1PreReqObjects)
	if err != nil {
		t.Fatalf("Error creating clients for %s: %v", packageName, err)
	}

	// Create Installation and reconcile on it.
	installation := &v1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-installation",
			Namespace: integreatlyOperatorNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Spec: v1alpha1.InstallationSpec{
			MasterURL:        "https://console.apps.example.com",
			RoutingSubdomain: "apps.example.com",
		},
	}
	ctx := context.TODO()
	testReconciler, err := NewReconciler(configManager, installation, fakeAppsV1Client, fakeOauthClient, fakeThreeScaleClient, mpm)
	status, err := testReconciler.Reconcile(ctx, installation, fakeSigsClient)
	if err != nil {
		t.Fatalf("Error reconciling %s: %v", packageName, err)
	}

	if status != v1alpha1.PhaseCompleted {
		t.Fatalf("unexpected status: %v, expected: %v", status, v1alpha1.PhaseCompleted)
	}

	err = assertInstallationSuccessfullyReconciled(installation, configManager, fakeSigsClient, fakeThreeScaleClient, fakeOauthClient)
	if err != nil {
		t.Fatal(err.Error())
	}

	// system-app and system-sidekiq deploymentconfigs should have been rolled out on first reconcile.
	sa, err := fakeAppsV1Client.DeploymentConfigs(defaultInstallationNamespace).Get("system-app", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting deplymentconfig: %v", err)
	}
	if sa.Status.LatestVersion == 1 {
		t.Fatalf("system-app was not rolled out")
	}
	ssk, err := fakeAppsV1Client.DeploymentConfigs(defaultInstallationNamespace).Get("system-sidekiq", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting deplymentconfig: %v", err)
	}
	if ssk.Status.LatestVersion == 1 {
		t.Fatalf("system-sidekiq was not rolled out")
	}

	// Ensure reconcile is idempotent
	installation.Status = integreatlyv1alpha1.InstallationStatus{
		ProductStatus: map[integreatlyv1alpha1.ProductName]string{
			integreatlyv1alpha1.Product3Scale: string(status),
		},
	}
	status, err = testReconciler.Reconcile(ctx, installation, fakeSigsClient)
	if err != nil {
		t.Fatalf("Error repeating reconciling %s: %v", packageName, err)
	}

	if status != v1alpha1.PhaseCompleted {
		t.Fatalf("unexpected status when repeating reconcile: %v, expected: %v", status, v1alpha1.PhaseCompleted)
	}

	err = assertInstallationSuccessfullyReconciled(installation, configManager, fakeSigsClient, fakeThreeScaleClient, fakeOauthClient)
	if err != nil {
		t.Fatal(err.Error())
	}

}
