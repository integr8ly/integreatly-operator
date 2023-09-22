package addon

import (
	"context"
	"net/http"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type deleteRHMIHandler struct {
	decoder    *admission.Decoder
	restConfig *rest.Config
	scheme     *runtime.Scheme
}

var _ admission.Handler = &deleteRHMIHandler{}
var _ admission.DecoderInjector = &deleteRHMIHandler{}

func NewDeleteRHMIHandler(config *rest.Config, scheme *runtime.Scheme) admission.Handler {
	return &deleteRHMIHandler{
		restConfig: config,
		scheme:     scheme,
	}
}

func (h *deleteRHMIHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *deleteRHMIHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	rhmi := &integreatlyv1alpha1.RHMI{}
	if err := h.decoder.DecodeRaw(request.OldObject, rhmi); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if len(rhmi.Finalizers) != 0 {
		return admission.Allowed("RHMI CR has finalizers")
	}

	return admission.Allowed("Operator Uninstalled")
}
