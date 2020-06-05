package common

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	goctx "context"
)

const (
	RHMIConfigCRName = "rhmi-config"
)

// we reuse this struct in tests A21 and A22
type MaintenanceBackup struct {
	Backup      v1alpha1.Backup
	Maintenance v1alpha1.Maintenance
}

// this state check covers test case - A22
// verify that the RHMIConfig validation webhook for Maintenance and Backup values work as expected
var maintenanceBackupStates = map[MaintenanceBackup]func(*testing.T) func(error){
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

var upgradeSectionStates = map[v1alpha1.Upgrade]func(*testing.T) func(error){
	{}: assertNoError,

	{
		NotBeforeDays: "-1",
	}: assertValidationError,

	{
		NotBeforeDays: "seven",
	}: assertValidationError,

	{
		WaitForMaintenance: "not even a boolean",
	}: assertValidationError,

	{
		NotBeforeDays:      "7",
		WaitForMaintenance: "true",
	}: assertNoError,
}

// TestRHMIConfigCRs tests that the RHMIConfig CR is created successfuly and
// validated.
func TestRHMIConfigCRs(t *testing.T, ctx *TestingContext) {
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
	verifyRHMIConfigMutatingWebhook(ctx, t)

	// Test each possible state for the Upgrade section
	for state, assertion := range upgradeSectionStates {
		t.Logf("Testing the RHMIConfig state: %s", logUpgrade(state))
		verifyRHMIConfigValidation(ctx.Client, assertion(t), func(cr *v1alpha1.RHMIConfig) {
			cr.Spec.Upgrade = state
		})
	}

	// test for possible state changes for the Backup and Maintenance section
	for state, assertion := range maintenanceBackupStates {
		verifyRHMIConfigValidation(ctx.Client, assertion(t), func(cr *v1alpha1.RHMIConfig) {
			cr.Spec.Maintenance.ApplyFrom = state.Maintenance.ApplyFrom
			cr.Spec.Backup.ApplyOn = state.Backup.ApplyOn
		})
	}
}

func verifyCr(t *testing.T, ctx *TestingContext) {
	t.Log("Verify rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{}

	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace}, rhmiConfig)
	if err != nil {
		t.Fatalf("Failed to verify RHMI Config resource %v", err)
	}

	// The upgrade fields should default to false
	if rhmiConfig.Spec.Upgrade.WaitForMaintenance != "true" {
		t.Errorf("WaitForMaintenance should be true by default. Got %s",
			rhmiConfig.Spec.Upgrade.WaitForMaintenance)
	}
	if rhmiConfig.Spec.Upgrade.NotBeforeDays != "7" {
		t.Errorf("NotBeforeDays should be set to 7. Got %s",
			rhmiConfig.Spec.Upgrade.NotBeforeDays)
	}
}

func deleteRHMIConfigCR(t *testing.T, client dynclient.Client, cr *v1alpha1.RHMIConfig) {
	if err := client.Delete(goctx.TODO(), cr); err != nil {
		t.Errorf("Failed to delete the rhmi config")
	}
}

func verifyRHMIConfigValidation(client dynclient.Client, validateError func(error), mutateRHMIConfig func(*v1alpha1.RHMIConfig)) error {
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
	validateError(client.Update(goctx.TODO(), rhmiConfig))

	return nil
}

// verifyRHMIConfigMutatingWebhook tests the mutating webhook by logging in as
// a customer admin in the testing IDP and performing an update to the RHMIConfig
// instance, and checking that the webhooks adds the correct annotations
func verifyRHMIConfigMutatingWebhook(ctx *TestingContext, t *testing.T) {
	// Create the testing IdP
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Errorf("Error when creating testing IdP: %v", err)
		return
	}

	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Errorf("Error getting RHMI CR: %v", err)
		return
	}

	masterURL := rhmi.Spec.MasterURL

	oauthRoute := &routev1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{
		Name:      resources.OpenshiftOAuthRouteName,
		Namespace: resources.OpenshiftAuthenticationNamespace,
	}, oauthRoute); err != nil {
		t.Errorf("Error getting Openshift OAuth Route: %v", err)
		return
	}

	// Get customer admin tokens
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), "customer-admin-1", DefaultPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Errorf("error occured trying to get token : %v", err)
		return
	}
	t.Log("Retrieved customer admin tokens")
	openshiftClient := resources.NewOpenshiftClient(ctx.HttpClient, masterURL)

	// Get the current RHMIConfig instance
	rhmiConfig := &v1alpha1.RHMIConfig{}
	if err := ctx.Client.Get(
		goctx.TODO(),
		types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace},
		rhmiConfig,
	); err != nil {
		t.Errorf("Error getting RHMIConfig instance: %v", err)
		return
	}

	// Update a value in the instance
	// The `TypeMeta` field has to be set explicitely in order to send the
	// marshalled RHMIConfig directly to the API
	rhmiConfig.TypeMeta = v1.TypeMeta{
		APIVersion: "integreatly.org/v1alpha1",
		Kind:       "RHMIConfig",
	}
	rhmiConfig.Spec.Upgrade.WaitForMaintenance = "false"
	rhmiConfigChange, err := json.Marshal(rhmiConfig)
	if err != nil {
		t.Errorf("Error marshalling rhmiConfig: %v", err)
		return
	}

	path := fmt.Sprintf("/apis/integreatly.org/v1alpha1/namespaces/%s/rhmiconfigs/%s",
		RHMIOperatorNamespace,
		RHMIConfigCRName,
	)

	// Update the RHMIConfig instance as the customer-admin user
	if _, err := openshiftClient.DoOpenshiftPutRequest(path, rhmiConfigChange); err != nil {
		t.Errorf("Error updating RHMIConfig instance: %v", err)
		return
	}

	// Get the updated RHMIConfig instance
	if err := ctx.Client.Get(
		goctx.TODO(),
		types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace},
		rhmiConfig,
	); err != nil {
		t.Errorf("Error getting RHMIConfig instance: %v", err)
		return
	}

	// Verify the username is set in the annotations
	if rhmiConfig.Annotations["lastEditUsername"] != "customer-admin-1" {
		t.Errorf("Expected mutating webhook to add lastEditUsername annotation to RHMIConfig. Got %s instead",
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

func assertNoError(t *testing.T) func(error) {
	return func(err error) {
		if err != nil {
			t.Errorf("Expected error to be nil. Got %v", err)
		}
	}
}

func assertValidationError(t *testing.T) func(error) {
	return func(err error) {
		switch e := err.(type) {
		case errors.APIStatus:
			if e.Status().Code != 403 {
				t.Errorf("Expected error to be \"Forbidden\", but got: %s", e.Status().Reason)
			}
		default:
			t.Errorf("Expected error type to be APIStatus type. Got %v", e)
		}
	}
}

func logUpgrade(upgrade v1alpha1.Upgrade) string {
	return fmt.Sprintf(
		"{ notBeforeDays: %s, waitForMaintenance: %s }",
		upgrade.NotBeforeDays,
		upgrade.WaitForMaintenance,
	)
}
