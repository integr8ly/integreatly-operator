package amqstreams

import (
	"context"
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

var (
	 installationNamespace string = "openshift-amq-streams"
	 installationName      string = "amq-streams-install-"
	 cvsName               string = "strimzi-cluster-operator.v0.11.1"
)

func NewReconciler(client pkgclient.Client, configManager config.ConfigReadWriter) (*Reconciler, error) {
	config, err := configManager.ReadAMQStreams()
	if err != nil {
		return nil, err
	}
	return &Reconciler{client: client, ConfigManager: configManager, Config: config}, nil
}

type Reconciler struct {
	client        pkgclient.Client
	Config        *config.AMQStreams
	ConfigManager config.ConfigReadWriter
}

func (r *Reconciler) Reconcile(phase v1alpha1.StatusPhase) (v1alpha1.StatusPhase, error) {
	switch phase {
	case v1alpha1.PhaseNone:
		return r.handleNoPhase()
	case v1alpha1.PhaseAccepted:
		return r.handleAcceptedPhase()
	case v1alpha1.PhaseAwaitingNS:
		return r.handleAwaitingNSPhase()
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase()
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do a dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of AMQ Streams failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for AMQ Streams: " + string(phase))
	}
}

func (r *Reconciler) handleNoPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams no phase")
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: installationNamespace,
			Name:      installationNamespace,
		},
	}
	err := r.client.Create(context.TODO(), ns)
	if err != nil && ! k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleAwaitingNSPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("waiting for namespace to be active")
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installationNamespace,
		},
	}
	err := r.client.Get(context.TODO(), pkgclient.ObjectKey{Name: installationNamespace}, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if ns.Status.Phase == v1.NamespaceActive {
		return v1alpha1.PhaseAccepted, nil
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleAcceptedPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams accepted phase")
	operatorGroup := &coreosv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: installationNamespace,
			GenerateName: installationName,
		},
		Spec: coreosv1.OperatorGroupSpec{
			TargetNamespaces: []string{installationNamespace},
		},
	}
	err := r.client.Create(context.TODO(), operatorGroup)
	subscription := &coreosv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: installationNamespace,
			GenerateName: installationName,
		},
		Spec: &coreosv1alpha1.SubscriptionSpec{
			InstallPlanApproval: coreosv1alpha1.ApprovalManual,
			StartingCSV: cvsName,
			Channel: "dev",
			Package: "integreatly",
		},
	}
	err = r.client.Create(context.TODO(), subscription)

	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams progress phase")

	return v1alpha1.PhaseCompleted, nil
}
