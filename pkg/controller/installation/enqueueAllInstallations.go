package installation

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/types"

	"github.com/sirupsen/logrus"

	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type EnqueueAllInstallations struct {
}

func (e *EnqueueAllInstallations) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.enqueueAllInstallations(q)
}

func (e *EnqueueAllInstallations) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.enqueueAllInstallations(q)
}

func (e *EnqueueAllInstallations) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.enqueueAllInstallations(q)
}

func (e *EnqueueAllInstallations) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.enqueueAllInstallations(q)
}

func (e *EnqueueAllInstallations) enqueueAllInstallations(q workqueue.RateLimitingInterface) {

	logrus.Info("user change triggered installation queue")

	// new client to avoid caching issues
	restConfig := controllerruntime.GetConfigOrDie()
	c, _ := k8sclient.New(restConfig, k8sclient.Options{})
	instList := &v1alpha1.InstallationList{}

	err := c.List(context.TODO(), instList)
	if err != nil {
		logrus.Errorf("error listing installations: %s", err.Error())
		return
	}

	for _, inst := range instList.Items {
		logrus.Infof("adding installation '%s/%s' to queue", inst.Namespace, inst.Name)
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: inst.Namespace, Name: inst.Name}})
	}
}
