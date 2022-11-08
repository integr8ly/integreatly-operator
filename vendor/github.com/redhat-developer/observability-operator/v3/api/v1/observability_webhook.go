/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var observabilitylog = logf.Log.WithName("observability-resource")

func (in *Observability) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:verbs=update,admissionReviewVersions=v1,sideEffects=None,path=/validate-observability-redhat-com-v1-observability,mutating=false,failurePolicy=fail,groups=observability.redhat.com,resources=observabilities,versions=v1,name=vobservability.kb.io

var _ webhook.Validator = &Observability{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Observability) ValidateCreate() error {
	observabilitylog.Info("validate create", "name", in.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Observability) ValidateUpdate(old runtime.Object) error {
	observabilitylog.Info("validate update", "name", in.Name)

	// For each value the following cannot be done
	// unset it if it's already present
	//	// set it if it's not set - the default kafka entry is already used
	//	// change it if it's set already
	// cannot remove the self contained block if it contained the value

	oldObsSpec := &old.(*Observability).Spec
	newObsSpec := &in.Spec

	//AlertManagerDefaultName
	if oldObsSpec.AlertManagerDefaultName != "" &&
		newObsSpec.AlertManagerDefaultName == "" {
		return errors.New("cannot unset AlertManagerDefaultName after cr creation")
	}

	if oldObsSpec.AlertManagerDefaultName == "" &&
		newObsSpec.AlertManagerDefaultName != "" {
		return errors.New("cannot set AlertManagerDefaultName after cr creation")
	}

	if oldObsSpec.AlertManagerDefaultName != "" &&
		newObsSpec.AlertManagerDefaultName != "" &&
		strings.Compare(oldObsSpec.AlertManagerDefaultName, newObsSpec.AlertManagerDefaultName) != 0 {
		return errors.New("cannot update AlertManagerDefaultName after cr creation")
	}

	//GrafanaDefaultName
	if oldObsSpec.GrafanaDefaultName != "" &&
		newObsSpec.GrafanaDefaultName == "" {
		return errors.New("cannot unset GrafanaDefaultName after cr creation")
	}

	if oldObsSpec.GrafanaDefaultName == "" &&
		newObsSpec.GrafanaDefaultName != "" {
		return errors.New("cannot set GrafanaDefaultName after cr creation")
	}

	if oldObsSpec.GrafanaDefaultName != "" &&
		newObsSpec.GrafanaDefaultName != "" &&
		strings.Compare(oldObsSpec.GrafanaDefaultName, newObsSpec.GrafanaDefaultName) != 0 {
		return errors.New("cannot update GrafanaDefaultName after cr creation")
	}

	//PrometheusDefaultName
	if oldObsSpec.PrometheusDefaultName != "" &&
		newObsSpec.PrometheusDefaultName == "" {
		return errors.New("cannot unset PrometheusDefaultName after cr creation")
	}

	if oldObsSpec.PrometheusDefaultName == "" &&
		newObsSpec.PrometheusDefaultName != "" {
		return errors.New("cannot set PrometheusDefaultName after cr creation")
	}

	if oldObsSpec.PrometheusDefaultName != "" &&
		newObsSpec.PrometheusDefaultName != "" &&
		strings.Compare(oldObsSpec.PrometheusDefaultName, newObsSpec.PrometheusDefaultName) != 0 {
		return errors.New("cannot update PrometheusDefaultName after cr creation")
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Observability) ValidateDelete() error {
	observabilitylog.Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
