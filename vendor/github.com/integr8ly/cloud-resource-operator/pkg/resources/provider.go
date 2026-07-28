package resources

import (
	"context"
	"reflect"

	"github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"

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

// resourceTypeSpecFromObject extracts ResourceTypeSpec from a CR Spec field.
// Spec may be ResourceTypeSpec itself (Postgres, BlobStorage) or embed it (Redis).
func resourceTypeSpecFromObject(o client.Object) (*croType.ResourceTypeSpec, error) {
	specField := reflect.ValueOf(o).Elem().FieldByName("Spec")
	if !specField.IsValid() {
		return nil, errors.New("object has no Spec field")
	}

	rtsType := reflect.TypeOf(croType.ResourceTypeSpec{})
	if specField.Type() == rtsType {
		v := specField.Interface().(croType.ResourceTypeSpec)
		return &v, nil
	}

	embedded := specField.FieldByName("ResourceTypeSpec")
	if embedded.IsValid() && embedded.Type() == rtsType {
		v := embedded.Interface().(croType.ResourceTypeSpec)
		return &v, nil
	}

	rts := &croType.ResourceTypeSpec{}
	if err := runtime.Field(reflect.ValueOf(o).Elem(), "Spec", rts); err != nil {
		return nil, err
	}
	return rts, nil
}

func (r *ReconcileResourceProvider) ReconcileResultSecret(ctx context.Context, o client.Object, d map[string][]byte) error {
	obj := o.(metav1.Object)
	secNs := obj.GetNamespace()
	rts, err := resourceTypeSpecFromObject(o)
	if err != nil {
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
	_, err = controllerruntime.CreateOrUpdate(ctx, r.Client, sec, func() error {
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

func IsLastResource(ctx context.Context, c client.Client) (bool, error) {
	listOptions := client.ListOptions{
		Namespace: "",
	}
	var postgresList = &v1alpha1.PostgresList{}
	if err := c.List(ctx, postgresList, &listOptions); err != nil {
		msg := "failed to retrieve postgres cr(s)"
		return false, errors.Wrap(err, msg)
	}
	if len(postgresList.Items) > 1 {
		return false, nil
	}
	var redisList = &v1alpha1.RedisList{}
	if err := c.List(ctx, redisList, &listOptions); err != nil {
		msg := "failed to retrieve redis cr(s)"
		return false, errors.Wrap(err, msg)
	}
	return (len(postgresList.Items) + len(redisList.Items)) == 1, nil
}
