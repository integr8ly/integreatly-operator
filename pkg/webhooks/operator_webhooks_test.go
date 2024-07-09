package webhooks

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	defaultNamespace = "redhat-rhmi-operator"
)

func TestReconcile(t *testing.T) {
	// Set up testing scheme
	testScheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	testScheme.AddKnownTypes(schemeBuilder.GroupVersion, &mockValidator{})

	if err := os.Setenv("WATCH_NAMESPACE", defaultNamespace); err != nil {
		t.Fatal(err)
	}

	// Create testing webhook config
	settings := IntegreatlyWebhookConfig{
		Enabled:     true,
		scheme:      testScheme,
		Port:        8090,
		CAConfigMap: "test-configmap",
		Webhooks: []IntegreatlyWebhook{
			{
				Name: "test",
				Rule: NewRule().
					OneResource("example.org", "v1", "mockvalidator").
					ForCreate().
					ForUpdate().
					NamespacedScope(),
				Register: ObjectWebhookRegister{
					&mockValidator{},
				},
			},
			{
				Name: "test-manual",
				Rule: NewRule().
					OneResource("example.org", "v1", "mockvalidator").
					ForCreate().
					ForUpdate().
					NamespacedScope(),
				Register: AdmissionWebhookRegister{
					Type: MutatingType,
					Path: "/mutate-me",
					Hook: &admission.Webhook{
						Handler: admission.HandlerFunc(func(ctx context.Context, req admission.Request) admission.Response {
							return admission.Patched("Updated")
						}),
					},
				},
			},
		},
	}

	rhmi := &v1alpha1.RHMI{}
	vwconf := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test.integreatly.org",
			Namespace: defaultNamespace,
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name: "test-manual",
				ClientConfig: admissionv1.WebhookClientConfig{
					CABundle: []byte("TEST"),
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: nil,
						Rule: admissionv1.Rule{
							APIGroups:   []string{"example.org"},
							APIVersions: []string{"v1"},
							Resources:   []string{"mockvalidator"},
						},
					},
				},
			},
		},
	}

	client := utils.NewTestClient(testScheme, rhmi, vwconf)

	// Start mock of CA controller
	done := make(chan struct{})
	defer close(done)
	go mockCAController(context.TODO(), client, done)

	// Perform one reconcilliation. After this, the ValidatingWebhookConfiguration
	// must have been created with the specification of the testing webhook
	if err := settings.Reconcile(context.TODO(), client, rhmi); err != nil {
		t.Fatalf("Error reconciling webhook objects: %v", err)
	}

	secret := &corev1.Secret{}
	if err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: "rhmi-webhook-cert", Namespace: defaultNamespace}, secret); err != nil {
		t.Errorf("Secret with TlS certs not found")
	} else {
		if string(secret.Data["tls.key"]) != "TLS KEY" {
			t.Errorf("Invalid value for secret tls.key. Expected TLS KEY, got %s", string(secret.Data["tls.key"]))
		}
		if string(secret.Data["tls.crt"]) != "TLS CERT" {
			t.Errorf("Invalid value for secret tls.crt. Expected TLS CERT, got %s", string(secret.Data["tls.crt"]))
		}
	}

	vwc, err := findValidatingWebhookConfig(client)
	if err != nil {
		t.Fatalf("Error finding ValidatingWebhookConfig: %v", err)
	}

	if len(vwc.Webhooks) != 1 {
		t.Fatalf("Expected one webhook to be registered, found %d", len(vwc.Webhooks))
	}

	webhook := vwc.Webhooks[0]

	if string(webhook.ClientConfig.CABundle) != "TEST" {
		t.Errorf("Expected CABundle field to be obtained from ConfigMap, but got %s", string(webhook.ClientConfig.CABundle))
	}

	if len(webhook.Rules) != 1 {
		t.Fatalf("Expected one rule to be registered, found %d", len(webhook.Rules))
	}

	rule := webhook.Rules[0]

	if rule.APIGroups[0] != "example.org" {
		t.Errorf("Expected rule.APIGroups to be [\"example.org\"], got %s", rule.APIGroups[0])
	}
	if rule.APIVersions[0] != "v1" {
		t.Errorf("Expected rule.APIVersions to be [\"v1\"], got %s", rule.APIVersions[0])
	}
	if rule.Resources[0] != "mockvalidator" {
		t.Errorf("Expected rule.Resources to be [\"mockvalidator\"], got %s", rule.Resources[0])
	}

	mwc, err := findMutatingWebhookConfig(client, "test")
	if err != nil {
		t.Fatalf("Error finding MutatingWebhookConfig 'test'")
	}

	if len(mwc.Webhooks) != 1 {
		t.Fatalf("Expected one webhook to be registered, found %d", len(mwc.Webhooks))
	}

	mutatingWebhook := mwc.Webhooks[0]

	if string(mutatingWebhook.ClientConfig.CABundle) != "TEST" {
		t.Errorf("Expected CABundle field to be obtained from ConfigMap, but got %s", string(webhook.ClientConfig.CABundle))
	}

	rule = mutatingWebhook.Rules[0]

	if rule.APIGroups[0] != "example.org" {
		t.Errorf("Expected rule.APIGroups to be [\"example.org\"], got %s", rule.APIGroups[0])
	}
	if rule.APIVersions[0] != "v1" {
		t.Errorf("Expected rule.APIVersions to be [\"v1\"], got %s", rule.APIVersions[0])
	}
	if rule.Resources[0] != "mockvalidator" {
		t.Errorf("Expected rule.Resources to be [\"mockvalidator\"], got %s", rule.Resources[0])
	}

	mwc, err = findMutatingWebhookConfig(client, "test-manual")
	if err != nil {
		t.Fatalf("Error finding MutatingWebhookConfig 'test-manual'")
	}

	if len(mwc.Webhooks) != 1 {
		t.Fatalf("Expected one webhook to be registered, found %d", len(mwc.Webhooks))
	}

	mutatingWebhook = mwc.Webhooks[0]

	if string(mutatingWebhook.ClientConfig.CABundle) != "TEST" {
		t.Errorf("Expected CABundle field to be obtained from ConfigMap, but got %s", string(webhook.ClientConfig.CABundle))
	}

	rule = mutatingWebhook.Rules[0]

	if rule.APIGroups[0] != "example.org" {
		t.Errorf("Expected rule.APIGroups to be [\"example.org\"], got %s", rule.APIGroups[0])
	}
	if rule.APIVersions[0] != "v1" {
		t.Errorf("Expected rule.APIVersions to be [\"v1\"], got %s", rule.APIVersions[0])
	}
	if rule.Resources[0] != "mockvalidator" {
		t.Errorf("Expected rule.Resources to be [\"mockvalidator\"], got %s", rule.Resources[0])
	}
}

type mockValidator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Value string
}

var _ admission.CustomValidator = &mockValidator{}
var _ runtime.Object = &mockValidator{}

func (m *mockValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	if m.Value == "correct" {
		return nil, nil
	}

	return nil, fmt.Errorf("Unexpected value. Expected correct, got %s", m.Value)
}

func (m *mockValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	if m.Value == "correct" {
		return nil, nil
	}

	return nil, fmt.Errorf("Unexpected value. Expected correct, got %s", m.Value)
}

func (m *mockValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, fmt.Errorf("Delete not allowed")
}

func (m *mockValidator) DeepCopyObject() runtime.Object {
	return &mockValidator{
		Value: m.Value,
	}
}

func (m *mockValidator) Default() {
}

var schemeBuilder = &scheme.Builder{
	GroupVersion: schema.GroupVersion{
		Group:   "example.org",
		Version: "v1",
	},
}

// Mock the behaviour of the CA controller, that injects a `service-ca.crt` field
// on ConfigMaps that are annotated with a specific annotation. This is used
// to obtain the CA that signs the certificates used by the webhook server and
// reference it in the ValidatingWebhookConfig CR
func mockCAController(ctx context.Context, client k8sclient.Client, stop <-chan struct{}) {
	for {
		// Stop if the channel was closed
		select {
		case <-stop:
			break
		default:
		}

		// Get the list of config maps
		configMaps := &corev1.ConfigMapList{}
		err := client.List(ctx, configMaps,
			k8sclient.InNamespace(defaultNamespace))
		if err != nil {
			continue
		}

		for _, configMap := range configMaps.Items {
			annotation, ok := configMap.Annotations[caConfigMapAnnotation]
			if !ok || annotation != "true" {
				continue
			}

			configMap.Data = map[string]string{
				"service-ca.crt": "TEST",
			}

			if err := client.Update(ctx, &configMap); err != nil {
				continue
			}
		}

		// Get the list of services
		services := &corev1.ServiceList{}
		if err := client.List(ctx, services,
			k8sclient.InNamespace(defaultNamespace)); err != nil {
			continue
		}

		for _, service := range services.Items {
			secretName, ok := service.Annotations[caServiceAnnotation]
			if !ok {
				continue
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: defaultNamespace,
				},
				Data: map[string][]byte{
					"tls.key": []byte("TLS KEY"),
					"tls.crt": []byte("TLS CERT"),
				},
			}

			if err := client.Create(ctx, secret); err != nil {
				continue
			}
		}
	}
}

func findValidatingWebhookConfig(client k8sclient.Client) (*admissionv1.ValidatingWebhookConfiguration, error) {
	vwc := &admissionv1.ValidatingWebhookConfiguration{}
	err := client.Get(
		context.TODO(),
		k8sclient.ObjectKey{Name: "test.integreatly.org", Namespace: defaultNamespace},
		vwc,
	)
	if err != nil {
		return nil, err
	}

	return vwc, nil
}

func findMutatingWebhookConfig(client k8sclient.Client, name string) (*admissionv1.MutatingWebhookConfiguration, error) {
	mwc := &admissionv1.MutatingWebhookConfiguration{}
	err := client.Get(
		context.TODO(),
		k8sclient.ObjectKey{Name: fmt.Sprintf("%s.integreatly.org", name)},
		mwc,
	)
	if err != nil {
		return nil, err
	}

	return mwc, nil
}
