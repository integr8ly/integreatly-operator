package common

import (
	"context"
	"time"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/controllers/status"
	addonv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	projectv1 "github.com/openshift/api/project/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStatusConditions(t TestingTB, ctx *TestingContext) {

	installed, err := status.IsAddonOperatorInstalled(ctx.Client)
	if err != nil {
		t.Fatal(err)
	}

	if !installed {
		t.Skip("addon operator not installed - skipping test")
	}

	testInitialInstalledHealthyNotDegraded(t, ctx)
	testNotHealthyAndDegraded(t, ctx)
	testHealthyAndDegraded(t, ctx)
}

func testInitialInstalledHealthyNotDegraded(t TestingTB, ctx *TestingContext) {
	t.Log("testInstalledHealthyNotDegraded")

	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 10*time.Minute, true, func(ctx2 context.Context) (done bool, err error) {
		addonInstance, err := getAddonInstance(ctx)
		if err != nil {
			t.Logf("unable to get addon instance: %v", err)
			return false, nil
		}

		// Check heart beat healthy - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionHealthy.String()) {
			t.Log("expected heart beat condition to be true but was false")
			return false, nil
		}

		// Check installed - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionInstalled.String()) {
			t.Log("expected installed condition to be true but was false")
			return false, nil
		}

		// Check degraded - false
		if !meta.IsStatusConditionFalse(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionDegraded.String()) {
			t.Log("expected degraded condition to be false but was true")
			return false, nil
		}

		// Check integreatly core components healthy - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, v1alpha1.HealthyConditionType.String()) {
			t.Log("expected core components healthy condition to be true but was false")
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		t.Fatal(err)
	}

}

func testNotHealthyAndDegraded(t TestingTB, ctx *TestingContext) {
	t.Log("testNotHealthyAndDegraded")
	// delete user sso operator namespace - core component
	deleteProject(t, ctx.Client, RHSSOUserOperatorNamespace)

	// After this test - ensure status is back to normal before progressing to the next test as it could make it flaky
	defer testInitialInstalledHealthyNotDegraded(t, ctx)

	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 10*time.Minute, true, func(ctx2 context.Context) (done bool, err error) {
		addonInstance, err := getAddonInstance(ctx)
		if err != nil {
			t.Logf("unable to get addon instance: %v", err)
			return false, nil
		}

		// Check heart beat healthy - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionHealthy.String()) {
			t.Log("expected heart beat condition to be true but was false")
			return false, nil
		}

		// Check installed - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionInstalled.String()) {
			t.Log("expected installed condition to be true but was false")

			return false, nil
		}

		// Check degraded - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionDegraded.String()) {
			t.Log("expected degraded condition to be true but was false")
			return false, nil
		}

		// Check integreatly core components healthy - false
		if !meta.IsStatusConditionFalse(addonInstance.Status.Conditions, v1alpha1.HealthyConditionType.String()) {
			t.Log("expected core components healthy condition to be false but was true")
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

func testHealthyAndDegraded(t TestingTB, ctx *TestingContext) {
	t.Log("testHealthyAndDegraded")
	// delete rhsso operator namespace - not a core component
	deleteProject(t, ctx.Client, Marin3rOperatorNamespace)

	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 10*time.Minute, true, func(ctx2 context.Context) (done bool, err error) {
		addonInstance, err := getAddonInstance(ctx)
		if err != nil {
			t.Logf("unable to get addon instance: %v", err)
			return false, nil
		}

		// Check heart beat healthy - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionHealthy.String()) {
			t.Log("expected heart beat condition to be true but was false")
			return false, nil
		}

		// Check installed - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionInstalled.String()) {
			t.Log("expected installed condition to be true but was false")

			return false, nil
		}

		// Check degraded - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, addonv1alpha1.AddonInstanceConditionDegraded.String()) {
			t.Log("expected degraded condition to be true but was false")
			return false, nil
		}

		// Check integreatly core components healthy - true
		if !meta.IsStatusConditionTrue(addonInstance.Status.Conditions, v1alpha1.HealthyConditionType.String()) {
			t.Log("expected core components healthy condition to be true but was false")
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

func getAddonInstance(ctx *TestingContext) (*addonv1alpha1.AddonInstance, error) {
	addonInstance := addonv1alpha1.AddonInstance{}
	if err := ctx.Client.Get(context.Background(), client.ObjectKey{Namespace: RHOAMOperatorNamespace, Name: "addon-instance"}, &addonInstance); err != nil {
		return nil, err
	}

	return &addonInstance, nil
}

func deleteProject(t TestingTB, client client.Client, name string) {
	if err := client.Delete(context.Background(), &projectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: name}}); err != nil {
		t.Fatal(err)
	}
	t.Logf("deleted %s namespace", name)
}
