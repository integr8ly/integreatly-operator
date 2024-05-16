package webhooks

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"os"
	"strings"
	"time"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	pkgerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// IntegreatlyWebhookConfig contains the data and logic to setup the webhooks
// server of a given Manager implementation, and to reconcile webhook configuration
// CRs pointing to the server.
type IntegreatlyWebhookConfig struct {
	scheme *runtime.Scheme

	Enabled bool

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

	// Register for the webhook into the server
	Register WebhookRegister
}

const (
	operatorPodServiceName = "rhmi-webhooks"
	operatorPodPort        = 8090
	servicePort            = 443
	mountedCertDir         = "/etc/ssl/certs/webhook"
	caConfigMap            = "rhmi-operator-ca"
	caConfigMapAnnotation  = "service.beta.openshift.io/inject-cabundle"
	caServiceAnnotation    = "service.beta.openshift.io/serving-cert-secret-name"
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
	if !webhookConfig.enabled() {
		return nil
	}

	// Create a new client to reconcile the Service. `mgr.GetClient()` can't
	// be used as it relies on the cache that hasn't been initialized yet
	client, err := k8sclient.New(mgr.GetConfig(), k8sclient.Options{
		Scheme: mgr.GetScheme(),
	})
	if err != nil {
		return err
	}

	// Create the service pointing to the operator pod
	if err := webhookConfig.ReconcileService(context.TODO(), client, nil); err != nil {
		return err
	}
	// Get the secret with the certificates for the service
	if err := webhookConfig.setupCerts(context.TODO(), client); err != nil {
		return err
	}

	// Create a webhook server.
	webhookServer := webhook.NewServer(webhook.Options{
		Port:    webhookConfig.Port,
		CertDir: webhookConfig.CertDir,
	})
	if err := mgr.Add(webhookServer); err != nil {
		return err
	}

	bldr := builder.WebhookManagedBy(mgr)

	for _, webhook := range webhookConfig.Webhooks {
		bldr = webhook.Register.RegisterToBuilder(bldr)
		webhook.Register.RegisterToServer(webhookServer)
	}

	err = bldr.Complete()
	if err != nil {
		return err
	}

	return nil
}

// Reconcile reconciles a `ValidationWebhookConfiguration` object for each webhook
// in `webhookConfig.Webhooks`, using the rules and the path as it's generated
// by controler-runtime webhook builder.
// It reconciles a Service that exposes the webhook server
// A ownerRef to the owner parameter is set on the reconciled resources. This
// parameter is optional, if `nil` is passed, no ownerReference will be set
func (webhookConfig *IntegreatlyWebhookConfig) Reconcile(ctx context.Context, client k8sclient.Client, owner ownerutil.Owner) error {
	if !webhookConfig.enabled() {
		return nil
	}

	watchNS, err := k8s.GetWatchNamespace()
	if err != nil {
		return pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"

	// Reconcile the Service
	if err := webhookConfig.ReconcileService(ctx, client, owner); err != nil {
		return err
	}

	// Create (if it doesn't exist) the config map where the CA certificate is
	// injected
	caConfigMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      webhookConfig.CAConfigMap,
			Namespace: namespacePrefix + "operator",
			Annotations: map[string]string{
				caConfigMapAnnotation: "true",
			},
		},
	}

	err = client.Create(ctx, caConfigMap)
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
		reconciler, err := webhook.Register.GetReconciler(webhookConfig.scheme)
		if err != nil {
			return err
		}

		reconciler.SetName(webhook.Name)
		reconciler.SetRule(webhook.Rule)

		if err := reconciler.Reconcile(ctx, client, caBundle); err != nil {
			return err
		}
	}

	return nil
}

// ReconcileService creates or updates the service that points to the Pod
func (webhookConfig *IntegreatlyWebhookConfig) ReconcileService(ctx context.Context, client k8sclient.Client, owner ownerutil.Owner) error {
	watchNS, err := k8s.GetWatchNamespace()
	if err != nil {
		return pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	// Get the service. If it's not found, create it
	service := &corev1.Service{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: namespacePrefix + "operator",
		Name:      operatorPodServiceName,
	}, service); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		return createService(ctx, client, owner)
	}

	// If the existing service has a different .spec.clusterIP value, delete it
	if service.Spec.ClusterIP != "None" {
		if err := client.Delete(ctx, service); err != nil {
			return err
		}
	}

	return createService(ctx, client, owner)
}

func createService(ctx context.Context, client k8sclient.Client, owner ownerutil.Owner) error {
	watchNS, err := k8s.GetWatchNamespace()
	if err != nil {
		return pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	service := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      operatorPodServiceName,
			Namespace: namespacePrefix + "operator",
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		if owner != nil {
			ownerutil.EnsureOwner(service, owner)
		}

		if service.Annotations == nil {
			service.Annotations = map[string]string{}
		}
		service.Annotations[caServiceAnnotation] = "rhmi-webhook-cert"
		service.Spec.ClusterIP = "None"

		service.Spec.Selector = map[string]string{
			"name": "rhmi-operator",
		}

		service.Spec.Ports = []corev1.ServicePort{
			{
				Protocol:   corev1.ProtocolTCP,
				Port:       443,
				TargetPort: intstr.FromInt(8090),
			},
		}

		return nil
	})
	return err
}

// setupCerts waits for the secret created for the operator Service to exist, and
// when it's ready, extracts the certificates and saves them in webhookConfig.CertDir
func (webhookConfig *IntegreatlyWebhookConfig) setupCerts(ctx context.Context, client k8sclient.Client) error {
	watchNS, err := k8s.GetWatchNamespace()
	if err != nil {
		return pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"

	// Wait for the secret to te created
	secret := &corev1.Secret{}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*1, time.Second*30, true, func(ctx2 context.Context) (bool, error) {
		err := client.Get(ctx, k8sclient.ObjectKey{Namespace: namespacePrefix + "operator", Name: "rhmi-webhook-cert"}, secret)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	// Save the key
	if err := webhookConfig.saveCertFromSecret(secret.Data, "tls.key"); err != nil {
		return err
	}
	// Save the cert
	return webhookConfig.saveCertFromSecret(secret.Data, "tls.crt")
}

func (webhookConfig *IntegreatlyWebhookConfig) waitForCAInConfigMap(ctx context.Context, client k8sclient.Client) ([]byte, error) {
	var caBundle []byte
	watchNS, err := k8s.GetWatchNamespace()
	if err != nil {
		return nil, pkgerr.Wrap(err, "could not get watch namespace from operator_webhooks reconcile")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"

	err = wait.PollUntilContextTimeout(context.TODO(), time.Second, time.Second*30, true, func(ctx2 context.Context) (bool, error) {
		caConfigMap := &corev1.ConfigMap{}
		if err := client.Get(ctx,
			k8sclient.ObjectKey{Name: webhookConfig.CAConfigMap, Namespace: namespacePrefix + "operator"},
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

func (webhookConfig *IntegreatlyWebhookConfig) saveCertFromSecret(secretData map[string][]byte, fileName string) error {
	value, ok := secretData[fileName]
	if !ok {
		return fmt.Errorf("Secret does not contain key %s", fileName)
	}

	// Save the key
	f, err := os.Create(fmt.Sprintf("%s/%s", webhookConfig.CertDir, fileName))
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("os file closer: %v", err)
		}
	}(f)

	_, err = f.Write(value)
	return err
}

func (webhookConfig *IntegreatlyWebhookConfig) enabled() bool {
	return webhookConfig.Enabled
}
