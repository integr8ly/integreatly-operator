package webhooks

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// WebhookRegister knows how the register a webhoop into the server. Either by
// regstering to the WebhookBuilder or directly to the webhook server.
type WebhookRegister interface {
	GetPath(scheme *runtime.Scheme) (string, error)

	RegisterToBuilder(blrd *builder.WebhookBuilder) *builder.WebhookBuilder
	RegisterToServer(scheme *runtime.Scheme, srv *webhook.Server)
}

// ObjectWebhookRegister registers objects that implement either the `Validator`
// interface or the `Defaulting` interface into the WebhookBuilder
type ObjectWebhookRegister struct {
	Object runtime.Object
}

// GetPath creates the path for the webhook as implemented at controller-runtime/pkg/builder/webhook.go
// in order to match the path registered under the hood by the WebhookBuilder
func (vwr ObjectWebhookRegister) GetPath(scheme *runtime.Scheme) (string, error) {
	gvk, err := apiutil.GVKForObject(vwr.Object, scheme)
	if err != nil {
		return "", err
	}

	path := "/validate-" + strings.Replace(gvk.Group, ".", "-", -1) + "-" +
		gvk.Version + "-" + strings.ToLower(gvk.Kind)

	return path, nil
}

// RegisterToBuilder adds the object into the builder, which registers the webhook
// for the object into the webhook server
func (vwr ObjectWebhookRegister) RegisterToBuilder(bldr *builder.WebhookBuilder) *builder.WebhookBuilder {
	return bldr.For(vwr.Object)
}

// RegisterToServer does nothing, as the register is done by the builder
func (vwr ObjectWebhookRegister) RegisterToServer(_ *runtime.Scheme, _ *webhook.Server) {}

// AdmissionWebhookRegister registers a given webhook into a specific path.
// This allows a more low level alternative to the WebhookBuilder, as it can
// directly get access the the AdmissionReview object sent to the webhook.
type AdmissionWebhookRegister struct {
	Hook *admission.Webhook
	Path string
}

// GetPath simply returns the path of `awr`
func (awr AdmissionWebhookRegister) GetPath(_ *runtime.Scheme) (string, error) {
	return awr.Path, nil
}

// RegisterToBuilder does not mutate the WebhookBuilder
func (awr AdmissionWebhookRegister) RegisterToBuilder(bldr *builder.WebhookBuilder) *builder.WebhookBuilder {
	return bldr
}

// RegisterToServer regsiters the webhook to the path of `awr`
func (awr AdmissionWebhookRegister) RegisterToServer(scheme *runtime.Scheme, srv *webhook.Server) {
	awr.Hook.InjectScheme(scheme)
	srv.Register(awr.Path, awr.Hook)
}
