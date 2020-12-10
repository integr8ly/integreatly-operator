package addon

import (
	"context"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"net/http"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// UninstallOperator uninstalls the RHMI operator by deleting the subscription
// and CSV. If the subscription is not found, it doesn't do anything, as the
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

	log.Infof("Deleting subscription", l.Fields{"name": subscription.Name})

	// Declare the deleting subscription function
	deleteSubscription := func() error {
		return client.Delete(ctx, subscription)
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

	// If the CSV wasn't found, just delete the subscription
	if k8serr.IsNotFound(err) {
		return deleteSubscription()
	}

	log.Infof("Deleting operator CSV", l.Fields{"name": csv.Name})

	// Delete the CSV
	if err := client.Delete(ctx, csv); err != nil {
		return err
	}

	// Delete the subscription
	return deleteSubscription()
}

type deleteRHMIHandler struct {
	decoder    *admission.Decoder
	restConfig *rest.Config
	client     k8sclient.Client
}

var _ admission.Handler = &deleteRHMIHandler{}
var _ admission.DecoderInjector = &deleteRHMIHandler{}

func NewDeleteRHMIHandler(config *rest.Config) admission.Handler {
	return &deleteRHMIHandler{
		restConfig: config,
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
		return admission.Allowed("RHMI Has finalizers")
	}

	client, err := h.getClient()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if err := UninstallOperator(ctx, client, rhmi); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.Allowed("RHMI Uninstalled")
}

func (h *deleteRHMIHandler) getClient() (k8sclient.Client, error) {
	if h.client == nil {
		c, err := k8sclient.New(h.restConfig, k8sclient.Options{})
		if err != nil {
			return nil, err
		}
		h.client = c
	}

	return h.client, nil
}
