package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceReconciler struct {
	client pkgclient.Client
	logger *logrus.Entry
}

func NewNamespaceReconciler(client pkgclient.Client, logger *logrus.Entry) *NamespaceReconciler {
	return &NamespaceReconciler{client: client, logger: logger}
}

func (nr *NamespaceReconciler) Reconcile(ctx context.Context, ns *v1.Namespace, owner *v1alpha1.Installation) (*v1.Namespace, error) {

	if ns.Name == "" {
		return ns, errors.New("cannot reconcile namespace, it has no name")
	}
	err := nr.client.Get(ctx, pkgclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		if !errors2.IsNotFound(err) {
			return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve namespace: %s", ns.Name))
		}
		decorateNS(ns, owner)
		if err = nr.client.Create(ctx, ns); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not create namespace: %s", ns.Name))
		}
		return ns, nil
	}
	// ns exists so check it is our namespace
	if !NSIsOwnedBy(ns, owner) && ns.Status.Phase != v1.NamespaceTerminating {
		return nil, errors.New("existing namespace found with name " + ns.Name + " but it is not owned by the integreatly installation")
	}

	decorateNS(ns, owner)
	if err := nr.client.Update(ctx, ns); err != nil {
		return ns, errors.Wrap(err, "failed to update the ns definition ")
	}
	return ns, nil
}

func decorateNS(ns *v1.Namespace, install *v1alpha1.Installation) {
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ref := v12.NewControllerRef(install, v1alpha1.SchemaGroupVersionKind)
	ns.Labels["integreatly"] = "true"
	refExists := false
	for _, er := range ns.OwnerReferences {
		if er.Name == ref.Name {
			refExists = true
			break
		}
	}
	if !refExists {
		ns.OwnerReferences = append(ns.OwnerReferences, *ref)
	}
}

func NSIsOwnedBy(ns *v1.Namespace, owner *v1alpha1.Installation) bool {
	for _, or := range ns.OwnerReferences {
		if or.Name == owner.Name && or.APIVersion == owner.APIVersion {
			return true
		}
	}
	return false
}
