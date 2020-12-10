package installation

import (
	"errors"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	IntegreatlyOwnerNamespace = "integreatly-namespace"
	IntegreatlyOwnerName      = "integreatly-name"
)

type EnqueueIntegreatlyOwner struct {
	log l.Logger
}

func (e *EnqueueIntegreatlyOwner) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if req, err := e.getIntegreatlyOwner(evt.Meta); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if req, err := e.getIntegreatlyOwner(evt.MetaNew); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if req, err := e.getIntegreatlyOwner(evt.Meta); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if req, err := e.getIntegreatlyOwner(evt.Meta); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) getIntegreatlyOwner(object metav1.Object) (reconcile.Request, error) {
	typeObj, _ := meta.TypeAccessor(object)
	ant := object.GetAnnotations()
	if ns, ok := ant[IntegreatlyOwnerNamespace]; ok {
		if name, ok := ant[IntegreatlyOwnerName]; ok {
			log.Info(fmt.Sprintf("%s %s/%s > got integreatly owner %s/%s", typeObj.GetKind(), object.GetNamespace(), object.GetName(), ns, name))
			return reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      name,
				},
			}, nil
		}
	}
	return reconcile.Request{}, errors.New("object does not have an integreatly owner")
}
