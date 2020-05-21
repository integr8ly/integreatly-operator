package common

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

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
	RHMIConfigCRName = "rhmi-config-test"
)

var upgradeSectionStates = map[v1alpha1.Upgrade]func(*testing.T) func(error){
	{}: assertNoError,

	{
		ApplyOn:               "",
		AlwaysImmediately:     true,
		DuringNextMaintenance: true,
	}: assertNoError,

	{
		ApplyOn:               "malformed date!",
		AlwaysImmediately:     false,
		DuringNextMaintenance: false,
	}: assertValidationError,

	// Valid: future date
	{
		ApplyOn:               time.Now().Add(time.Hour).UTC().Format("2 Jan 2006 15:04"),
		AlwaysImmediately:     false,
		DuringNextMaintenance: false,
	}: assertNoError,

	// Invalid: past date
	{
		ApplyOn:               time.Now().Add(-time.Hour).UTC().Format("2 Jan 2006 15:04"),
		AlwaysImmediately:     false,
		DuringNextMaintenance: false,
	}: assertValidationError,

	// Invalid: valid date, but `duringNextmaintenance` is set
	{
		ApplyOn:               time.Now().Add(time.Hour).UTC().Format("2 Jan 2006 15:04"),
		AlwaysImmediately:     false,
		DuringNextMaintenance: true,
	}: assertValidationError,

	// Invalid: valid date, but `alwaysImmediately` is set
	{
		ApplyOn:               time.Now().Add(time.Hour).UTC().Format("2 Jan 2006 15:04"),
		AlwaysImmediately:     true,
		DuringNextMaintenance: false,
	}: assertValidationError,
}

// TestRHMIConfigCRs tests that the RHMIConfig CR is created successfuly and
// validated.
func TestRHMIConfigCRs(t *testing.T, ctx *TestingContext) {
	t.Log("Test rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      RHMIConfigCRName,
			Namespace: RHMIOperatorNamespace,
		},
	}

	// Wait for the ValidatingWebhookConfiguration to be reconciled. In the edge
	// case that this test is run so fast that the operator mightn't have had
	// time to reconcile it.
	if err := waitForValidatingWebhook(ctx.Client); err != nil {
		t.Fatalf("Error waiting for ValidatingWebhookConfiguration: %v", err)
	}

	if err := ctx.Client.Create(goctx.TODO(), rhmiConfig); err != nil {
		t.Fatalf("Failed to create RHMI Config resource %v", err)
	}

	// Clean up after testing
	defer deleteRHMIConfigCR(t, ctx.Client, rhmiConfig)

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
}

func verifyCr(t *testing.T, ctx *TestingContext) {
	t.Log("Verify rhmi config cr creation")

	rhmiConfig := &v1alpha1.RHMIConfig{}

	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: RHMIConfigCRName, Namespace: RHMIOperatorNamespace}, rhmiConfig)
	if err != nil {
		t.Fatalf("Failed to verify RHMI Config resource %v", err)
	}

	// The upgrade fields should default to false
	if rhmiConfig.Spec.Upgrade.AlwaysImmediately != false {
		t.Errorf("AlwaysImmediately should be false by default")
	}
	if rhmiConfig.Spec.Upgrade.DuringNextMaintenance != false {
		t.Errorf("DuringNextMaintenance should be set to false")
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
// a customer admin in the testing IdP and performing an update to the RHMIConfig
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
	rhmiConfig.Spec.Upgrade.AlwaysImmediately = true
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
		"{ applyOn: %s, alwaysImmediately: %t, duringNextMaintenance: %t }",
		upgrade.ApplyOn,
		upgrade.AlwaysImmediately,
		upgrade.DuringNextMaintenance,
	)
}
