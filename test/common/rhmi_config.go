package common

import (
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	goctx "context"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RHMIConfigCRName = "rhmi-config"
	pollInterval     = 1 * time.Second
	pollTimeout      = 30 * time.Second
)

// we reuse this struct in tests A21 and A22
type MaintenanceBackup struct {
	Backup      v1alpha1.Backup
	Maintenance v1alpha1.Maintenance
}

// this state check covers test case - A22
// verify that the RHMIConfig validation webhook for Maintenance and Backup values work as expected
var maintenanceBackupStates = map[MaintenanceBackup]func(TestingTB) func(error) error{
	// we expect no error as blank strings will be set to default vals
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "",
		},
	}: assertNoError,
	// valid input format hh:mm and ddd hh:mm
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "20:05",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Sun 22:10",
		},
	}: assertNoError,
	// we expect an error due to both times being parsed as a 1 hour window
	// for aws these windows can not overlap
	// this state provides overlapping times
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "20:05",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Sun 20:15",
		},
	}: assertValidationError,
	// another overlap check, we want to ensure we get an error from a single minute overlap
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "20:15",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Thu 19:16",
		},
	}: assertValidationError,
	// we expect the following :
	//  * Backup hh:mm
	//  * Maintenance ddd hh:mm
	// the following checks will verify malformed times
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "26:00",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Sun 12:05",
		},
	}: assertValidationError,
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "22:00",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Malformed 12:05",
		},
	}: assertValidationError,
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "malformed",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Sun 20:00",
		},
	}: assertValidationError,
	{
		Backup: v1alpha1.Backup{
			ApplyOn: "20:00",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "malformed",
		},
	}: assertValidationError,
}

var upgradeSectionStates = map[v1alpha1.Upgrade]func(TestingTB) func(error) error{
	{}: assertNoError,

	{
		NotBeforeDays: intPtr(-1),
		Schedule:      boolPtr(true),
	}: assertValidationError,

	{
		NotBeforeDays:      intPtr(7),
		WaitForMaintenance: boolPtr(true),
		Schedule:           boolPtr(true),
	}: assertNoError,
}

// TestRHMIConfigCRs tests that the RHMIConfig CR is created successfuly and
// validated.
func TestRHMIConfigCRs(t TestingTB, ctx *TestingContext) {
	t.Log("Test rhmi config cr creation")

	rhmiConfig := RHMIConfigTemplate()

	// we need to delete the CR to ensure it is a blank CR for validation/creation
	deleteRHMIConfigCR(t, ctx.Client, rhmiConfig)

	// Wait for the ValidatingWebhookConfiguration to be reconciled. In the edge
	// case that this test is run so fast that the operator mightn't have had
	// time to reconcile it.
	if err := waitForValidatingWebhook(ctx.Client); err != nil {
		t.Fatalf("Error waiting for ValidatingWebhookConfiguration: %v", err)
	}

	if _, err := controllerutil.CreateOrUpdate(goctx.TODO(), ctx.Client, rhmiConfig, func() error {
		return nil
	}); err != nil {
		t.Fatalf("Failed to create RHMI Config resource %v", err)
	}

	// Test the CR is created with default values
	verifyCr(t, ctx)

	// Verify the Mutating webhook
	// Use polling to avoid unnecessary test failure due to an error
	// when trying to update modified object
	// More info in https://github.com/integr8ly/integreatly-operator/pull/1279
	err := wait.Poll(pollInterval, pollTimeout, func() (done bool, err error) {
		newErr := verifyRHMIConfigMutatingWebhook(ctx, t)
		if newErr != nil {
			return false, newErr
		}
		return true, nil
	})
	if err != nil {
		t.Errorf("Timed out when trying to verify RHMI Config Mutating Webhook: %s", err)
	}

	// Test each possible state for the Upgrade section
	for state, assertion := range upgradeSectionStates {
		t.Logf("Testing the RHMIConfig state: %s", logUpgrade(state))
		err := wait.Poll(pollInterval, pollTimeout, func() (done bool, err error) {
			newErr := verifyRHMIConfigValidation(ctx.Client, assertion(t), func(cr *v1alpha1.RHMIConfig) {
				cr.Spec.Upgrade = state
			})
			if newErr != nil {
				return false, newErr
			}
			return true, nil

		})
		if err != nil {
			t.Errorf("Timed out when trying to test states for the Upgrade section: %s", err)
		}

	}

	// test for possible state changes for the Backup and Maintenance section
	for state, assertion := range maintenanceBackupStates {

		err := wait.Poll(pollInterval, pollTimeout, func() (done bool, err error) {
			newErr := verifyRHMIConfigValidation(ctx.Client, assertion(t), func(cr *v1alpha1.RHMIConfig) {
				cr.Spec.Maintenance.ApplyFrom = state.Maintenance.ApplyFrom
				cr.Spec.Backup.ApplyOn = state.Backup.ApplyOn
			})
			if newErr != nil {
				return false, newErr
			}
			return true, nil
		})
		if err != nil {
			t.Errorf("Timed out when trying to test states for the maintenance and backup sections: %s", err)
		}
	}
}

func verifyCr(t TestingTB, ctx *TestingContext) {
	t.Log("Verify rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{}

	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace}, rhmiConfig)
	if err != nil {
		t.Fatalf("Failed to verify RHMI Config resource %v", err)
	}

	// The upgrade fields should default to false
	if *rhmiConfig.Spec.Upgrade.WaitForMaintenance != true {
		t.Errorf("WaitForMaintenance should be true by default. Got %v",
			rhmiConfig.Spec.Upgrade.WaitForMaintenance)
	}
	if *rhmiConfig.Spec.Upgrade.NotBeforeDays != 7 {
		t.Errorf("NotBeforeDays should be set to 7. Got %v",
			rhmiConfig.Spec.Upgrade.NotBeforeDays)
	}
}

func deleteRHMIConfigCR(t TestingTB, client dynclient.Client, cr *v1alpha1.RHMIConfig) {
	if err := client.Delete(goctx.TODO(), cr); err != nil {
		t.Errorf("Failed to delete the rhmi config")
	}
}

func verifyRHMIConfigValidation(client dynclient.Client, validateError func(error) error, mutateRHMIConfig func(*v1alpha1.RHMIConfig)) error {
	rhmiConfig := &v1alpha1.RHMIConfig{}

	if err := client.Get(
		goctx.TODO(),
		types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace},
		rhmiConfig,
	); err != nil {
		return err
	}

	// Perform the update and validate the error object
	mutateRHMIConfig(rhmiConfig)
	return validateError(client.Update(goctx.TODO(), rhmiConfig))

}

// verifyRHMIConfigMutatingWebhook tests the mutating webhook by logging in as
// a customer admin in the testing IDP and performing an update to the RHMIConfig
// instance, and checking that the webhooks adds the correct annotations
func verifyRHMIConfigMutatingWebhook(ctx *TestingContext, t TestingTB) error {
	currentUser := &userv1.User{}
	if err := ctx.Client.Get(goctx.TODO(), dynclient.ObjectKey{
		Name: "~",
	}, currentUser); err != nil {
		t.Logf("Error getting the current user: %v", err)
		return err
	}

	// Get the current RHMIConfig instance
	rhmiConfig := &v1alpha1.RHMIConfig{}
	if err := ctx.Client.Get(
		goctx.TODO(),
		types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace},
		rhmiConfig,
	); err != nil {
		t.Logf("Error getting RHMIConfig instance: %v", err)
		return err
	}

	// Update a value in the instance
	*rhmiConfig.Spec.Upgrade.WaitForMaintenance = false

	// Update the RHMIConfig instance as the customer-admin user
	if err := ctx.Client.Update(goctx.TODO(), rhmiConfig); err != nil {
		t.Logf("Error updating RHMIConfig instance: %v", err)
		return err
	}

	// Get the updated RHMIConfig instance
	if err := ctx.Client.Get(
		goctx.TODO(),
		types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace},
		rhmiConfig,
	); err != nil {
		t.Logf("Error getting RHMIConfig instance: %v", err)
		return err
	}

	// Verify the username is set in the annotations
	if rhmiConfig.Annotations["lastEditUsername"] != currentUser.Name {
		t.Errorf("Expected mutating webhook to add \"%s\" lastEditUsername annotation to RHMIConfig. Got %s instead",
			currentUser.Name,
			rhmiConfig.Annotations["lastEditUsername"])
	}

	// Verify the timestamp is set in the annotations
	if lastEdit, ok := rhmiConfig.Annotations["lastEditTimestamp"]; ok {
		if _, err := time.Parse("2 Jan 2006 15:04", lastEdit); err != nil {
			t.Errorf("Expected lastEditTimestamp to be parsed, but got error: %v", err)
		}
	} else {
		t.Error("Expected mutating webhook to add lastEditTimestamp annotation to RHMIConfig")
	}
	return nil
}

// we require this template across different tests for RHMI config
// rhmi config we need to use the config map provisioned in the RHMI install
// this to avoid a conflict of having multiple rhmi configs
func RHMIConfigTemplate() *v1alpha1.RHMIConfig {
	return &v1alpha1.RHMIConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      RHMIConfigCRName,
			Namespace: RHMIOperatorNamespace,
		},
	}
}

func waitForValidatingWebhook(client dynclient.Client) error {
	return wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		vwc := &v1beta1.ValidatingWebhookConfiguration{}
		err := client.Get(goctx.TODO(),
			dynclient.ObjectKey{Name: "rhmiconfig.integreatly.org"},
			vwc,
		)
		if err == nil {
			return true, nil
		}
		if errors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	})
}

func assertNoError(t TestingTB) func(error) error {
	return func(err error) error {
		if err != nil {
			t.Logf("Expected error to be nil. Got %v", err)
			return err
		}
		return nil
	}
}

func assertValidationError(t TestingTB) func(error) error {
	return func(err error) error {
		switch e := err.(type) {
		case errors.APIStatus:
			if e.Status().Code != 403 {
				t.Logf("Expected error to be \"Forbidden\", but got: %s", e.Status().Reason)
				return err
			}
		default:
			t.Logf("Expected error type to be APIStatus type. Got %v", e)
			return err
		}
		return nil
	}
}

func logUpgrade(upgrade v1alpha1.Upgrade) string {
	return fmt.Sprintf(
		"{ notBeforeDays: %v, waitForMaintenance: %v }",
		upgrade.NotBeforeDays,
		upgrade.WaitForMaintenance,
	)
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
