package amqstreams

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/sirupsen/logrus"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase()
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do r dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of AMQ Streams failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for AMQ Streams: " + string(phase))
	}
}

func (r *Reconciler) handleNoPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams no phase")
	return v1alpha1.PhaseAccepted, nil
}

func (r *Reconciler) handleAcceptedPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams accepted phase")
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams progress phase")
	return v1alpha1.PhaseCompleted, nil
}
