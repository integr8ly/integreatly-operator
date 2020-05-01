package installation

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	olmv1alpha1.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

const (
	defaultNamespace = "redhat-rhmi-operator"
)

func TestGetIntegreatlyOperatorSubscription(t *testing.T) {
	nonRhmiSubscriptionMock := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "another-subscription",
			Namespace: defaultNamespace,
		},
	}

	rhmiSubscriptionMock := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redhat-rhmi-subscription",
			Namespace: defaultNamespace,
		},
	}

	subscriptionListMock := &olmv1alpha1.SubscriptionList{
		Items: []olmv1alpha1.Subscription{
			*nonRhmiSubscriptionMock,
			*rhmiSubscriptionMock,
		},
	}

	scenarios := []struct {
		Name       string
		FakeClient k8sclient.Client
		Context    context.Context
		Verify     func(rhmiSubscription *olmv1alpha1.Subscription, err error)
	}{
		{
			Name:       "Test rhmi subscription not retrieved",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme()),
			Context:    context.TODO(),
			Verify: func(rhmiSubscription *olmv1alpha1.Subscription, err error) {
				// Should return a not found error with no rhmi subscription present
				if !k8serr.IsNotFound(err) {
					t.Fatalf("Unexpected error %v", err)
				}
			},
		},
		{
			Name:       "Test rhmi subscription not retrieved with other subscriptions in namespace",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme(), nonRhmiSubscriptionMock),
			Verify: func(rhmiSubscription *olmv1alpha1.Subscription, err error) {
				// Should return a not found error with no rhmi subscription present
				if !k8serr.IsNotFound(err) {
					t.Fatalf("Unexpected error %v", err)
				}
			},
		},
		{
			Name:       "Test rhmi subscription retrieved",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme(), rhmiSubscriptionMock),
			Verify: func(rhmiSubscription *olmv1alpha1.Subscription, err error) {
				// No errors return when RHMI subscription found
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				if !reflect.DeepEqual(rhmiSubscription, rhmiSubscriptionMock) {
					t.Fatalf("Non RHMI subscription returned %v", err)
				}
			},
		},
		{
			Name:       "Test rhmi subscription retrieved with other subscriptions in namespace",
			FakeClient: fake.NewFakeClientWithScheme(buildScheme(), subscriptionListMock),
			Verify: func(rhmiSubscription *olmv1alpha1.Subscription, err error) {
				// No errors return when RHMI subscription found
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				if !reflect.DeepEqual(rhmiSubscription, rhmiSubscriptionMock) {
					t.Fatalf("Non RHMI subscription returned %v", err)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			r := &ReconcileInstallation{client: scenario.FakeClient, context: scenario.Context}
			rhmiSubscription, err := r.getIntegreatlyOperatorSubscription(defaultNamespace)

			scenario.Verify(rhmiSubscription, err)
		})
	}
}

func TestIsRhmiUpgradeAvailable(t *testing.T) {
	scenarios := []struct {
		Name                           string
		RhmiSubscription               *olmv1alpha1.Subscription
		ExpectedUpgradeAvailableResult bool
	}{
		{
			Name:                           "Test no subscription found",
			RhmiSubscription:               nil,
			ExpectedUpgradeAvailableResult: false,
		},
		{
			Name: "Test same RHMI CSV version in subscription",
			RhmiSubscription: &olmv1alpha1.Subscription{
				Status: olmv1alpha1.SubscriptionStatus{
					CurrentCSV:   "1.0.0",
					InstalledCSV: "1.0.0",
				},
			},
			ExpectedUpgradeAvailableResult: false,
		},
		{
			Name: "Test new RHMI CSV version in subscription",
			RhmiSubscription: &olmv1alpha1.Subscription{
				Status: olmv1alpha1.SubscriptionStatus{
					CurrentCSV:   "1.0.1",
					InstalledCSV: "1.0.0",
				},
			},
			ExpectedUpgradeAvailableResult: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			r := &ReconcileInstallation{}

			upgradeAvailable := r.isRHMIUpgradeAvailable(scenario.RhmiSubscription)
			if upgradeAvailable != scenario.ExpectedUpgradeAvailableResult {
				t.Fatalf("Expected upgradeAvailable to be %v but got %v", scenario.ExpectedUpgradeAvailableResult, upgradeAvailable)
			}
		})
	}
}

func TestIsRhmiUpgradeServiceAffecting(t *testing.T) {
	scenarios := []struct {
		Name                           string
		RhmiCSV                        *olmv1alpha1.ClusterServiceVersion
		ExpectedServiceAffectingResult bool
	}{
		{
			Name:                           "Test no CSV",
			RhmiCSV:                        nil,
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with no service_affecting annotation",
			RhmiCSV: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with service_affecting annotation true",
			RhmiCSV: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"serviceAffecting": "true",
					},
				},
			},
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with service_affecting annotation false",
			RhmiCSV: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"serviceAffecting": "false",
					},
				},
			},
			ExpectedServiceAffectingResult: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			r := &ReconcileInstallation{}

			isServiceAffecting := r.isRHMIUpgradeServiceAffecting(scenario.RhmiCSV)
			if isServiceAffecting != scenario.ExpectedServiceAffectingResult {
				t.Fatalf("Expected isServiceAffecting to be %v but got %v", scenario.ExpectedServiceAffectingResult, isServiceAffecting)
			}
		})
	}
}

func TestApproveRHMIUpgrade(t *testing.T) {
	installPlanObjectMeta := metav1.ObjectMeta{
		Name:      "rhmi-ip",
		Namespace: defaultNamespace,
	}

	installPlanReadyForApproval := &olmv1alpha1.InstallPlan{
		ObjectMeta: installPlanObjectMeta,
		Spec: olmv1alpha1.InstallPlanSpec{
			Approved: false,
			ClusterServiceVersionNames: []string{
				"RHMI-v1.0.0",
			},
		},
	}

	installPlanAlreadyUpgrading := &olmv1alpha1.InstallPlan{
		ObjectMeta: installPlanObjectMeta,
		Spec: olmv1alpha1.InstallPlanSpec{
			Approved: false,
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Phase: olmv1alpha1.InstallPlanPhaseInstalling,
		},
	}

	scenarios := []struct {
		Name            string
		FakeClient      k8sclient.Client
		Context         context.Context
		EventRecorder   record.EventRecorder
		RhmiInstallPlan *olmv1alpha1.InstallPlan
		Verify          func(rhmiInstallPlan *olmv1alpha1.InstallPlan, err error)
	}{
		{
			Name:            "Test install plan already upgrading",
			FakeClient:      fake.NewFakeClientWithScheme(buildScheme(), installPlanAlreadyUpgrading),
			Context:         context.TODO(),
			EventRecorder:   setupRecorder(),
			RhmiInstallPlan: installPlanAlreadyUpgrading,
			Verify: func(updatedRhmiInstallPlan *olmv1alpha1.InstallPlan, err error) {
				// Should not return an error
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				if updatedRhmiInstallPlan.Spec.Approved != false {
					t.Fatalf("Expected installPlan to not be upgraded")
				}
			},
		},
		{
			Name:            "Test install plan ready to upgrade",
			FakeClient:      fake.NewFakeClientWithScheme(buildScheme(), installPlanReadyForApproval),
			Context:         context.TODO(),
			EventRecorder:   setupRecorder(),
			RhmiInstallPlan: installPlanReadyForApproval,
			Verify: func(updatedRhmiInstallPlan *olmv1alpha1.InstallPlan, err error) {
				// Should not return an error
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				if updatedRhmiInstallPlan.Spec.Approved != true {
					t.Fatalf("Expected installplan.Spec.Approved to be true")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			r := &ReconcileInstallation{client: scenario.FakeClient, context: scenario.Context}

			r.approveRHMIUpgrade(scenario.RhmiInstallPlan, scenario.EventRecorder)

			retrievedInstallPlan := &olmv1alpha1.InstallPlan{}
			err := r.client.Get(scenario.Context, k8sclient.ObjectKey{Name: scenario.RhmiInstallPlan.Name, Namespace: scenario.RhmiInstallPlan.Namespace}, retrievedInstallPlan)

			scenario.Verify(retrievedInstallPlan, err)
		})
	}
}

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
	installation, err := getInstallationCR(ctx, mockClient, t)
	assertCRValue(installation.Spec.UseClusterStorage)

	return nil
}

// Utility function to retrieve the Installation CR
func getInstallationCR(ctx context.Context, serverClient k8sclient.Client, t *testing.T) (*integreatlyv1alpha1.RHMI, error) {
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
		t.Fatal("More than one installation found")
		return nil, nil
	}

	return &installationList.Items[0], nil
}
