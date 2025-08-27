package rhmiConfigs

import (
	"context"
	"fmt"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/utils"
	"github.com/integr8ly/integreatly-operator/version"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultNamespace = "testing-namespaces-operator"
)

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestIsUpgradeAvailable(t *testing.T) {
	scenarios := []struct {
		Name                           string
		RhoamSubscription              *operatorsv1alpha1.Subscription
		ExpectedUpgradeAvailableResult bool
	}{
		{
			Name:                           "Test no subscription found",
			RhoamSubscription:              nil,
			ExpectedUpgradeAvailableResult: false,
		},
		{
			Name: "Test same RHOAM CSV version in subscription",
			RhoamSubscription: &operatorsv1alpha1.Subscription{
				Status: operatorsv1alpha1.SubscriptionStatus{
					CurrentCSV:   "1.0.0",
					InstalledCSV: "1.0.0",
				},
			},
			ExpectedUpgradeAvailableResult: false,
		},
		{
			Name: "Test new RHOAM CSV version in subscription",
			RhoamSubscription: &operatorsv1alpha1.Subscription{
				Status: operatorsv1alpha1.SubscriptionStatus{
					CurrentCSV:   "1.0.1",
					InstalledCSV: "1.0.0",
				},
			},
			ExpectedUpgradeAvailableResult: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			upgradeAvailable := IsUpgradeAvailable(scenario.RhoamSubscription)
			if upgradeAvailable != scenario.ExpectedUpgradeAvailableResult {
				t.Fatalf("Expected upgradeAvailable to be %v but got %v", scenario.ExpectedUpgradeAvailableResult, upgradeAvailable)
			}
		})
	}
}

func TestIsUpgradeServiceAffecting(t *testing.T) {
	scenarios := []struct {
		Name                           string
		RhoamCSV                       *operatorsv1alpha1.ClusterServiceVersion
		ExpectedServiceAffectingResult bool
	}{
		{
			Name:                           "Test no CSV",
			RhoamCSV:                       nil,
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with no service_affecting annotation",
			RhoamCSV: &operatorsv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with service_affecting annotation true",
			RhoamCSV: &operatorsv1alpha1.ClusterServiceVersion{
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
			RhoamCSV: &operatorsv1alpha1.ClusterServiceVersion{
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
			isServiceAffecting := IsUpgradeServiceAffecting(scenario.RhoamCSV)
			if isServiceAffecting != scenario.ExpectedServiceAffectingResult {
				t.Fatalf("Expected isServiceAffecting to be %v but got %v", scenario.ExpectedServiceAffectingResult, isServiceAffecting)
			}
		})
	}
}

func TestApproveUpgrade(t *testing.T) {
	installPlanObjectMeta := metav1.ObjectMeta{
		Name:      "rhoam-ip",
		Namespace: defaultNamespace,
	}

	installPlanReadyForApproval := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: installPlanObjectMeta,
		Spec: operatorsv1alpha1.InstallPlanSpec{
			Approved: false,
			ClusterServiceVersionNames: []string{
				"RHOAM-v1.0.0",
			},
		},
		Status: operatorsv1alpha1.InstallPlanStatus{
			Plan: []*operatorsv1alpha1.Step{
				{
					Resource: operatorsv1alpha1.StepResource{
						Kind:     "ClusterServiceVersion",
						Manifest: fmt.Sprintf("{\"kind\":\"ClusterServiceVersion\",    \"spec\": {      \"version\": \"%s\"}}", version.GetVersion()),
					},
				},
			},
		},
	}

	rhoamMock := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhoam",
			Namespace: defaultNamespace,
		},
	}

	installPlanAlreadyUpgrading := &operatorsv1alpha1.InstallPlan{
		ObjectMeta: installPlanObjectMeta,
		Spec: operatorsv1alpha1.InstallPlanSpec{
			Approved: false,
		},
		Status: operatorsv1alpha1.InstallPlanStatus{
			Phase: operatorsv1alpha1.InstallPlanPhaseInstalling,
		},
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name            string
		FakeClient      k8sclient.Client
		Context         context.Context
		EventRecorder   record.EventRecorder
		RhmiInstallPlan *operatorsv1alpha1.InstallPlan
		RHMI            *integreatlyv1alpha1.RHMI
		Verify          func(rhmiInstallPlan *operatorsv1alpha1.InstallPlan, rhmi *integreatlyv1alpha1.RHMI, err error)
	}{
		{
			Name:            "Test install plan already upgrading",
			FakeClient:      utils.NewTestClient(scheme, installPlanAlreadyUpgrading, rhoamMock),
			Context:         context.TODO(),
			EventRecorder:   setupRecorder(),
			RhmiInstallPlan: installPlanAlreadyUpgrading,
			RHMI:            rhoamMock,
			Verify: func(updatedRhmiInstallPlan *operatorsv1alpha1.InstallPlan, rhmi *integreatlyv1alpha1.RHMI, err error) {
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
			FakeClient:      utils.NewTestClient(scheme, installPlanReadyForApproval, rhoamMock),
			Context:         context.TODO(),
			EventRecorder:   setupRecorder(),
			RhmiInstallPlan: installPlanReadyForApproval,
			RHMI:            rhoamMock,
			Verify: func(updatedRhmiInstallPlan *operatorsv1alpha1.InstallPlan, rhmi *integreatlyv1alpha1.RHMI, err error) {
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
			err := ApproveUpgrade(context.TODO(), scenario.FakeClient, scenario.RHMI, scenario.RhmiInstallPlan, scenario.EventRecorder)
			if err != nil {
				t.Fatal(err)
			}
			retrievedInstallPlan := &operatorsv1alpha1.InstallPlan{}
			err = scenario.FakeClient.Get(scenario.Context, k8sclient.ObjectKey{Name: scenario.RhmiInstallPlan.Name, Namespace: scenario.RhmiInstallPlan.Namespace}, retrievedInstallPlan)
			if err != nil {
				t.Fatal(err)
			}
			rhmi := &integreatlyv1alpha1.RHMI{}
			err = scenario.FakeClient.Get(scenario.Context, k8sclient.ObjectKey{Name: scenario.RHMI.Name, Namespace: scenario.RHMI.Namespace}, rhmi)
			scenario.Verify(retrievedInstallPlan, rhmi, err)
		})
	}
}
