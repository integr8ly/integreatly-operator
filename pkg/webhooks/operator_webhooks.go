package webhooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IntegreatlyWebhookConfig contains the data and logic to setup the webhooks
// server of a given Manager implementation, and to reconcile ValidatingWebhookConfiguration
// CRs pointing to the server.
type IntegreatlyWebhookConfig struct {
	scheme *runtime.Scheme

	Port        int
	CertDir     string
	CAConfigMap string

	Webhooks []IntegreatlyWebhook
}

// IntegreatlyWebhook acts as a single source of truth for validating webhooks
// managed by the operator. It's data are used both for registering the
// endpoing to the webhook server and to reconcile the ValidatingWebhookConfiguration
// that points to the server.
type IntegreatlyWebhook struct {
	// Name of the webhook. Used to generate a name for the ValidatingWebhookConfiguration
	Name string

	// Rule for the webhook to be triggered
	Rule RuleWithOperations

	// Implementation of the `Validator` interface that performs the validation
	Validator admission.Validator
}

const (
	operatorPodServiceName = "rhmi-webhooks"
	operatorPodPort        = 8090
	servicePort            = 443
	mountedCertDir         = "/etc/ssl/certs/webhook"
	caConfigMap            = "rhmi-operator-ca"
	caConfigMapAnnotation  = "service.beta.openshift.io/inject-cabundle"
)

// Config is a global instance. The same instance is needed in order to use the
// same configuration for the webhooks server that's run at startup and the
// reconcilliation of the ValidatingWebhookConfiguration CRs
var Config *IntegreatlyWebhookConfig = &IntegreatlyWebhookConfig{
	// Port that the webhook service is pointing to
	Port: operatorPodPort,

	// Mounted as a volume from the secret generated from Openshift
	CertDir: mountedCertDir,

	// Name of the config map where the CA certificate is injected
	CAConfigMap: caConfigMap,

	// List of webhooks to configure
	Webhooks: []IntegreatlyWebhook{},
}

// SetupServer sets up the webhook server managed by mgr with the settings from
// webhookConfig. It sets the port and cert dir based on the settings and
// registers the Validator implementations from each webhook from webhookConfig.Webhooks
func (webhookConfig *IntegreatlyWebhookConfig) SetupServer(mgr manager.Manager) error {
	webhookServer := mgr.GetWebhookServer()
	webhookServer.Port = webhookConfig.Port
	webhookServer.CertDir = webhookConfig.CertDir

	webhookConfig.scheme = mgr.GetScheme()

	bldr := builder.WebhookManagedBy(mgr)

	for _, webhook := range webhookConfig.Webhooks {
		bldr = bldr.For(webhook.Validator)
	}

	bldr.Complete()

	return nil
}

// Reconcile reconciles a `ValidationWebhookConfiguration` object for each webhook
// in `webhookConfig.Webhooks`, using the rules and the path as it's generated
// by controler-runtime webhook builder.
// It assumes the injection of the CA that signs the TLS certificates into a ConfigMap
// to be stored in the `ValidationWebhookConfiguration`
func (webhookConfig *IntegreatlyWebhookConfig) Reconcile(ctx context.Context, client k8sclient.Client) error {
	// Create (if it doesn't exist) the config map where the CA certificate is
	// injected
	caConfigMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      webhookConfig.CAConfigMap,
			Namespace: "redhat-rhmi-operator",
			Annotations: map[string]string{
				caConfigMapAnnotation: "true",
			},
		},
	}

	err := client.Create(ctx, caConfigMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// Wait for the config map to be injected with the CA
	caBundle, err := webhookConfig.waitForCAInConfigMap(ctx, client)
	if err != nil {
		return err
	}

	// Reconcile the webhooks
	for _, webhook := range webhookConfig.Webhooks {
		err := webhookConfig.reconcileValidationWebhook(ctx, client, caBundle, webhook)
		if err != nil {
			return err
		}
	}

	return nil
}

func (webhookConfig *IntegreatlyWebhookConfig) reconcileValidationWebhook(ctx context.Context, client k8sclient.Client, caBundle []byte, webhook IntegreatlyWebhook) error {
	// We need to declare some parameters of the CR before as it expects pointers
	var (
		sideEffects    = v1beta1.SideEffectClassNone
		port           = int32(servicePort)
		matchPolicy    = v1beta1.Exact
		failurePolicy  = v1beta1.Fail
		timeoutSeconds = int32(30)
		path, err      = webhookConfig.getPath(webhook)
	)

	if err != nil {
		return err
	}

	// Get the ConfigMap where the CA certificate is injected
	caConfigMap := &corev1.ConfigMap{}
	if err := client.Get(ctx,
		k8sclient.ObjectKey{Name: webhookConfig.CAConfigMap, Namespace: "redhat-rhmi-operator"},
		caConfigMap,
	); err != nil {
		return err
	}

	// Create ValidatingWebhookConfiguration CR pointing to the webhook server
	cr := &v1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s.integreatly.org", webhook.Name),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, cr, func() error {
		cr.Webhooks = []v1beta1.ValidatingWebhook{
			{
				Name:        fmt.Sprintf("%s-validating-config.integreatly.org", webhook.Name),
				SideEffects: &sideEffects,
				ClientConfig: v1beta1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &v1beta1.ServiceReference{
						Namespace: "redhat-rhmi-operator",
						Name:      operatorPodServiceName,
						Path:      &path,
						Port:      &port,
					},
				},
				Rules: []v1beta1.RuleWithOperations{
					{
						Operations: webhook.Rule.Operations,
						Rule: v1beta1.Rule{
							APIGroups:   webhook.Rule.APIGroups,
							APIVersions: webhook.Rule.APIVersions,
							Resources:   webhook.Rule.Resources,
							Scope:       &webhook.Rule.Scope,
						},
					},
				},
				MatchPolicy:             &matchPolicy,
				AdmissionReviewVersions: []string{"v1beta1"},
				FailurePolicy:           &failurePolicy,
				TimeoutSeconds:          &timeoutSeconds,
			},
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (webhookConfig *IntegreatlyWebhookConfig) waitForCAInConfigMap(ctx context.Context, client k8sclient.Client) ([]byte, error) {
	var caBundle []byte

	err := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		caConfigMap := &corev1.ConfigMap{}
		if err := client.Get(ctx,
			k8sclient.ObjectKey{Name: webhookConfig.CAConfigMap, Namespace: "redhat-rhmi-operator"},
			caConfigMap,
		); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		result, ok := caConfigMap.Data["service-ca.crt"]

		if !ok {
			return false, nil
		}

		caBundle = []byte(result)
		return true, nil
	})

	return caBundle, err
}

// AddWebhook adds a webhook configuration to a webhookSettings. This must be done before
// starting the server as it registers the endpoints for the validation
func (webhookConfig *IntegreatlyWebhookConfig) AddWebhook(webhook IntegreatlyWebhook) {
	webhookConfig.Webhooks = append(webhookConfig.Webhooks, webhook)
}

// Copied from unexported implementation on controller-runtime/pkg/builder/webhook.go
func (webhookConfig *IntegreatlyWebhookConfig) getPath(webhook IntegreatlyWebhook) (string, error) {
	gvk, err := apiutil.GVKForObject(webhook.Validator, webhookConfig.scheme)
	if err != nil {
		return "", err
	}

	path := "/validate-" + strings.Replace(gvk.Group, ".", "-", -1) + "-" +
		gvk.Version + "-" + strings.ToLower(gvk.Kind)

	return path, nil
}
