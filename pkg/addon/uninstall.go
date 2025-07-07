package addon

import (
	"context"
	"net/http"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// UninstallOperator uninstalls the RHMI operator by deleting the CSV.
// If the subscription is not found, it doesn't do anything, as the
// operator might not be run through OLM
func UninstallOperator(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) error {
	// Get the operator subscription
	subscription, err := GetSubscription(ctx, client, installation)
	if err != nil {
		return err
	}

	// If the subscription is not found, finish: the operator might have been
	// running locally
	if subscription == nil {
		return nil
	}

	// Retrieve the operator CSV
	csv := &operatorsv1alpha1.ClusterServiceVersion{}
	err = client.Get(ctx, k8sclient.ObjectKey{
		Name:      subscription.Status.InstalledCSV,
		Namespace: installation.Namespace,
	}, csv)
	// If there's an unexpected error, return it
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}

	// If the CSV wasn't found, there is nothing left to delete
	if k8serr.IsNotFound(err) {
		return nil
	}

	log.Infof("Deleting operator CSV", l.Fields{"name": csv.Name})

	// Delete the CSV
	return client.Delete(ctx, csv)
}

type deleteRHMIHandler struct {
	decoder    *admission.Decoder
	restConfig *rest.Config
	scheme     *runtime.Scheme
	client     k8sclient.Client
}

var _ admission.Handler = &deleteRHMIHandler{}

//var _ admission.DecoderInjector = &deleteRHMIHandler{}

func NewDeleteRHMIHandler(config *rest.Config, scheme *runtime.Scheme, decoder *admission.Decoder) admission.Handler {
	return &deleteRHMIHandler{
		restConfig: config,
		scheme:     scheme,
		decoder:    decoder,
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

	client, err := h.getClient()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if err := UninstallOperator(ctx, client, rhmi); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.Allowed("Operator Uninstalled")
}

func (h *deleteRHMIHandler) getClient() (k8sclient.Client, error) {
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
