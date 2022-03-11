package addon

import (
	"context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ManagedAPIService = "managed-api-service"
	RHMI              = "rhmi"
)

var (
	addonNames = map[integreatlyv1alpha1.InstallationType]string{
		integreatlyv1alpha1.InstallationTypeManagedApi: ManagedAPIService,
		integreatlyv1alpha1.InstallationTypeManaged:    RHMI,
	}
	log = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "addon"})
)

// GetName resolves the add-on name given the installation type
func GetName(installationType integreatlyv1alpha1.InstallationType) string {
	addonName, ok := addonNames[installationType]
	if !ok {
		return RHMI
	}

	return addonName
}

//adding a few things here to managed 3scale CR webhook handler
type threescaleCRBlockCreateHandler struct {
	decoder    *admission.Decoder
	restConfig *rest.Config
	scheme     *runtime.Scheme
	client     k8sclient.Client
}

var _ admission.Handler = &threescaleCRBlockCreateHandler{}
var _ admission.DecoderInjector = &threescaleCRBlockCreateHandler{}

func New3scaleCRBlockCreateHandler(config *rest.Config, scheme *runtime.Scheme) admission.Handler {
	return &threescaleCRBlockCreateHandler{
		restConfig: config,
		scheme:     scheme,
	}
}

func (h *threescaleCRBlockCreateHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *threescaleCRBlockCreateHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	rhmi := &integreatlyv1alpha1.RHMI{}
	if err := h.decoder.DecodeRaw(request.Object, rhmi); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	//if err := h.decoder.DecodeRaw(request.OldObject, ns); err != nil {
	//	return admission.Errored(http.StatusBadRequest, err)
	//}

	// I have access to the CR so I can access the type so can trigger on that
	if (rhmi.Spec.Type) == "managed" || (rhmi.Spec.Type) == "managed-api" {
		return admission.Allowed("Allow CR creation on managed-api and managed types")
	}

	//must be a better way to get this but will do for now
	//if (ns.Name) == "sandbox-rhoam-3scale" {
	//	return admission.Allowed("Allow CR creation in sandbox-rhoam-3scale ns")
	//}
    // if none of the other conditions are met block creation
	return admission.Denied("Denied CR creation on multitenant-managed-api")
}

func (h *threescaleCRBlockCreateHandler) getClient() (k8sclient.Client, error) {
	if h.client == nil {
		c, err := k8sclient.New(h.restConfig, k8sclient.Options{
			Scheme: h.scheme,
		})
		if err != nil {
			return nil, err
		}
		h.client = c
	}

	return h.client, nil
}
