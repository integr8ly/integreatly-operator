package webapp

import (
	"context"
	"encoding/json"

	solutionExplorerv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/tutorial-web-app-operator/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/products/solutionexplorer"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type UpgradeNotifier interface {
	NotifyUpgrade(config *integreatlyv1alpha1.RHMIConfig, version string, isServiceAffecting bool) (integreatlyv1alpha1.StatusPhase, error)
	ClearNotification() error
}

type UpgradeNotifierImpl struct {
	client k8sclient.Client
	ctx    context.Context
}

func NewUpgradeNotifier(ctx context.Context, restConfig *rest.Config) (UpgradeNotifier, error) {
	client, err := k8sclient.New(restConfig, k8sclient.Options{})
	if err != nil {
		return nil, err
	}

	return NewUpgradeNotifierWithClient(ctx, client), nil
}

func NewUpgradeNotifierWithClient(ctx context.Context, client k8sclient.Client) *UpgradeNotifierImpl {
	return &UpgradeNotifierImpl{
		client: client,
		ctx:    ctx,
	}
}

type upgradeData struct {
	ScheduledFor       string `json:"scheduledFor"`
	Version            string `json:"version"`
	IsServiceAffecting bool   `json:"isServiceAffecting"`
}

func (notifier *UpgradeNotifierImpl) NotifyUpgrade(config *integreatlyv1alpha1.RHMIConfig, version string, isServiceAffecting bool) (integreatlyv1alpha1.StatusPhase, error) {
	webapp := &solutionExplorerv1alpha1.WebApp{
		ObjectMeta: v1.ObjectMeta{
			Name:      solutionexplorer.DefaultName,
			Namespace: "redhat-rhmi-solution-explorer",
		},
	}
	if err := notifier.client.Get(notifier.ctx, k8sclient.ObjectKey{
		Name:      webapp.Name,
		Namespace: webapp.Namespace,
	}, webapp); err != nil {
		if errors.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseInProgress, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Get the upgrade data
	upgrade := makeUpgradeData(config, version, isServiceAffecting)
	encoded, err := json.Marshal(upgrade)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Update the WebApp
	webapp.Spec.Template.Parameters[solutionexplorer.ParamUpgradeData] = string(encoded)
	if err := notifier.client.Update(notifier.ctx, webapp); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (notifier *UpgradeNotifierImpl) ClearNotification() error {
	webapp := &solutionExplorerv1alpha1.WebApp{
		ObjectMeta: v1.ObjectMeta{
			Name:      solutionexplorer.DefaultName,
			Namespace: "redhat-rhmi-solution-explorer",
		},
	}
	if err := notifier.client.Get(
		notifier.ctx,
		k8sclient.ObjectKey{Name: webapp.Name, Namespace: webapp.Namespace},
		webapp,
	); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	webapp.Spec.Template.Parameters[solutionexplorer.ParamUpgradeData] = "null"

	return notifier.client.Update(notifier.ctx, webapp)
}

func makeUpgradeData(rhmiConfig *integreatlyv1alpha1.RHMIConfig, version string, isServiceAffecting bool) *upgradeData {
	var scheduledFor string
	if rhmiConfig.Status.Upgrade.Scheduled != nil {
		scheduledFor = rhmiConfig.Status.Upgrade.Scheduled.For
	}

	return &upgradeData{
		ScheduledFor:       scheduledFor,
		Version:            version,
		IsServiceAffecting: isServiceAffecting,
	}
}

type NoOp struct {
}

func (noop *NoOp) NotifyUpgrade(config *integreatlyv1alpha1.RHMIConfig, version string, isServiceAffecting bool) (integreatlyv1alpha1.StatusPhase, error) {
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (noop *NoOp) ClearNotification() error {
	return nil
}
