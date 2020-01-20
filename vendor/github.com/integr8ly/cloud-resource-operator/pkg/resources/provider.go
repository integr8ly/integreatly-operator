package resources

import (
	"context"
	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"reflect"

	"github.com/pkg/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ReconcileResourceProvider struct {
	Client client.Client
	Scheme *runtime.Scheme
	Logger *logrus.Entry
}

func NewResourceProvider(c client.Client, s *runtime.Scheme, l *logrus.Entry) *ReconcileResourceProvider {
	return &ReconcileResourceProvider{
		Client: c,
		Scheme: s,
		Logger: l,
	}
}

func (r *ReconcileResourceProvider) ReconcileResultSecret(ctx context.Context, o runtime.Object, d map[string][]byte) error {
	obj := o.(metav1.Object)
	secNs := obj.GetNamespace()
	rts := &croType.ResourceTypeSpec{}
	if err := runtime.Field(reflect.ValueOf(o).Elem(), "Spec", rts); err != nil {
		return errors.Wrap(err, "failed to retrieve secret reference from instance")
	}
	if rts.SecretRef.Namespace != "" {
		secNs = rts.SecretRef.Namespace
	}
	sec := &v1.Secret{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      rts.SecretRef.Name,
			Namespace: secNs,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(ctx, r.Client, sec, func() error {
		if ownerRefErr := controllerutil.SetControllerReference(obj, sec, r.Scheme); ownerRefErr != nil {
			if updateErr := UpdatePhase(ctx, r.Client, o, croType.PhaseFailed, "setting secret data"); updateErr != nil {
				return updateErr
			}
			return errors.Wrapf(ownerRefErr, "failed to set owner on secret %s", sec.Name)
		}
		sec.Data = d
		sec.Type = v1.SecretTypeOpaque
		return nil
	})
	if err != nil {
		if updateErr := UpdatePhase(ctx, r.Client, o, croType.PhaseFailed, "failed to reconcile instance secret"); updateErr != nil {
			return updateErr
		}
		return errors.Wrapf(err, "failed to reconcile smtp credential set instance secret %s", sec.Name)
	}
	return nil
}
