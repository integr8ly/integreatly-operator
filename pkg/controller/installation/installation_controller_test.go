package installation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	olmv1alpha1.SchemeBuilder.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	return scheme
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

var (
	defaultNamespace = "testing-namespaces-operator"
)

// Test that the installation CR spec value for UseClusterStorage is true when the
// USE_CLUSTER_STORAGE environment variable is set to true
func TestCreateInstallationCR_useClusterStorage_true(t *testing.T) {
	err := testCreateInstallationCR_useClusterStorage(t, "true", func(useClusterStorage string) {
		if strings.ToLower(useClusterStorage) != "true" {
			t.Fatalf("Expected Installation.Spec.UseClusterStorage value to be true, but got %s", useClusterStorage)
		}
	})

	if err != nil {
		t.Fatalf("Error creating installation CR: %v", err)
	}
}

// Test that the installation CR spec value for UseClusterStorage is false when the
// USE_CLUSTER_STORAGE environment variable is set to false
func TestCreateInstallationCR_useClusterStorage_false(t *testing.T) {
	err := testCreateInstallationCR_useClusterStorage(t, "false", func(useClusterStorage string) {
		if strings.ToLower(useClusterStorage) != "false" {
			t.Fatalf("Expected Installation.Spec.UseClusterStorage value to be false, but got %s", useClusterStorage)
		}
	})

	if err != nil {
		t.Fatalf("Error creating Installation CR: %v", err)
	}
}

// Test that the installation CR spec value for UseClusterStorage is true when the
// USE_CLUSTER_STORAGE environment is not set
func TestCreateInstallationCR_useClusterStorage_default(t *testing.T) {
	testCreateInstallationCR_useClusterStorage(t, "", func(useClusterStorage string) {
		if useClusterStorage != "" {
			t.Fatalf("Expected default value of Installation.Spec.UseClusterStorage to be '', instead got %s", useClusterStorage)
		}
	})
}

// Utility higher order function to test the `createInstallationCR` function. Calls the function,
// retrieves the created Installation CR and delegates the assertion on a given function
func testCreateInstallationCR_useClusterStorage(t *testing.T, envValue string, assertCRValue func(useClusterStorage string)) error {
	mockClient := fake.NewFakeClientWithScheme(buildScheme())
	ctx := context.TODO()

	// Set USE_CLUSTER_STORAGE to true
	os.Setenv("USE_CLUSTER_STORAGE", envValue)
	os.Setenv("WATCH_NAMESPACE", defaultNamespace)

	// Defer unsetting the environment variables regardless of test results
	defer os.Unsetenv("USE_CLUSTER_STORAGE")
	defer os.Unsetenv("WATCH_NAMESPACE")

	// Function to test
	err := createInstallationCR(ctx, mockClient)

	if err != nil {
		return err
	}

	// Get the created installation and perform the assertion on it's
	// .Spec.UseClusterStorage value
	installation, err := getInstallationCR(ctx, mockClient)
	if err != nil {
		t.Fatalf("Error getting installation CR: %v", err)
	}

	assertCRValue(installation.Spec.UseClusterStorage)

	return nil
}

func TestCreateInstallationCR_alertingEmailAddressIsPresent(t *testing.T) {

	mockClient := fake.NewFakeClientWithScheme(buildScheme())
	ctx := context.TODO()

	email := "noreply-test@rhmi-redhat.com"
	buEmail := "noreply-test@rhmi-redhat.com"

	os.Setenv("ALERTING_EMAIL_ADDRESS", email)
	os.Setenv("BU_ALERTING_EMAIL_ADDRESS", buEmail)
	os.Setenv("WATCH_NAMESPACE", defaultNamespace)

	// Defer unsetting the environment variables regardless of test results
	defer os.Unsetenv("ALERTING_EMAIL_ADDRESS")
	defer os.Unsetenv("BU_ALERTING_EMAIL_ADDRESS")
	defer os.Unsetenv("WATCH_NAMESPACE")

	// Function to test
	err := createInstallationCR(ctx, mockClient)

	if err != nil {
		t.Fatalf("Error creating installation CR: %v", err)
	}

	installation, err := getInstallationCR(ctx, mockClient)
	if err != nil {
		t.Fatalf("Error getting installation CR: %v", err)
	}

	if installation.Spec.AlertingEmailAddresses.CSSRE != email {
		t.Fatalf(
			"Expected email address value of Installation.Spec.AlertingEmailAddresses.CSSRE to be %s, instead got %s",
			email,
			installation.Spec.AlertingEmailAddresses.CSSRE,
		)
	}

	if installation.Spec.AlertingEmailAddresses.BusinessUnit != buEmail {
		t.Fatalf(
			"Expected email address value of Installation.Spec.AlertingEmailAddresses.BusinessUnit to be %s, instead got %s",
			buEmail,
			installation.Spec.AlertingEmailAddresses.BusinessUnit,
		)
	}
}

func TestCreateInstallationCR_alertingEmailAddressIsNotPresent(t *testing.T) {

	mockClient := fake.NewFakeClientWithScheme(buildScheme())
	ctx := context.TODO()

	os.Setenv("WATCH_NAMESPACE", defaultNamespace)

	// Defer unsetting the environment variables regardless of test results
	defer os.Unsetenv("WATCH_NAMESPACE")

	// Function to test
	err := createInstallationCR(ctx, mockClient)
	if err != nil {
		t.Fatalf("Error creating installation CR: %v", err)
	}

	installation, err := getInstallationCR(ctx, mockClient)
	if err != nil {
		t.Fatalf("Error getting installation CR: %v", err)
	}

	if installation.Spec.AlertingEmailAddresses.CSSRE != "" {
		t.Fatalf(
			"Expected email address value of Installation.Spec.AlertingEmailAddresses.CSSRE to be empty, instead got %s",
			installation.Spec.AlertingEmailAddresses.CSSRE,
		)
	}

	if installation.Spec.AlertingEmailAddresses.BusinessUnit != "" {
		t.Fatalf(
			"Expected email address value of Installation.Spec.AlertingEmailAddresses.BusinessUnit to be empty, instead got %s",
			installation.Spec.AlertingEmailAddresses.BusinessUnit,
		)
	}
}

func TestCreateInstallationCR_installationTypeInEnvVar(t *testing.T) {

	mockClient := fake.NewFakeClientWithScheme(buildScheme())
	ctx := context.TODO()

	installationType := "test"
	os.Setenv("INSTALLATION_TYPE", installationType)
	os.Setenv("WATCH_NAMESPACE", defaultNamespace)

	// Defer unsetting the environment variables regardless of test results
	defer os.Unsetenv("INSTALLATION_TYPE")
	defer os.Unsetenv("WATCH_NAMESPACE")

	// Function to test
	err := createInstallationCR(ctx, mockClient)

	if err != nil {
		t.Fatalf("Error creating installation CR: %v", err)
	}

	installation, err := getInstallationCR(ctx, mockClient)
	if err != nil {
		t.Fatalf("Error getting installation CR: %v", err)
	}

	if installation.Spec.Type != installationType {
		t.Fatalf(
			"Expected installationType value of Installation.Spec.Type to be %s, instead got %s",
			installationType,
			installation.Spec.Type,
		)
	}
}

func TestCreateInstallationCR_installationTypeDefault(t *testing.T) {

	mockClient := fake.NewFakeClientWithScheme(buildScheme())
	ctx := context.TODO()

	installationType := "managed"
	os.Setenv("WATCH_NAMESPACE", defaultNamespace)

	// Defer unsetting the environment variables regardless of test results
	defer os.Unsetenv("WATCH_NAMESPACE")

	// Function to test
	err := createInstallationCR(ctx, mockClient)

	if err != nil {
		t.Fatalf("Error creating installation CR: %v", err)
	}

	installation, err := getInstallationCR(ctx, mockClient)
	if err != nil {
		t.Fatalf("Error getting installation CR: %v", err)
	}

	if installation.Spec.Type != installationType {
		t.Fatalf(
			"Expected installationType value of Installation.Spec.Type to be %s, instead got %s",
			installationType,
			installation.Spec.Type,
		)
	}
}

// Utility function to retrieve the Installation CR
func getInstallationCR(ctx context.Context, serverClient k8sclient.Client) (*integreatlyv1alpha1.RHMI, error) {
	namespace, err := k8sutil.GetWatchNamespace()

	if err != nil {
		return nil, err
	}

	installationList := &integreatlyv1alpha1.RHMIList{}
	listOps := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	err = serverClient.List(ctx, installationList, listOps...)

	if err != nil {
		return nil, err
	}

	if len(installationList.Items) == 0 {
		return nil, nil
	} else if len(installationList.Items) > 1 {
		return nil, fmt.Errorf("More than one installation found")
	}

	return &installationList.Items[0], nil
}
