/*
Copyright YEAR Red Hat, Inc.

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

package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RHMIConfigSpec defines the desired state of RHMIConfig
type RHMIConfigSpec struct {
	Upgrade     Upgrade     `json:"upgrade,omitempty"`
	Maintenance Maintenance `json:"maintenance,omitempty"`
	Backup      Backup      `json:"backup,omitempty"`
}

// RHMIConfigStatus defines the observed state of RHMIConfig
type RHMIConfigStatus struct {
	// status block reflects the current configuration of the cr
	//
	//	status:
	//		maintenance:
	//			apply-from: 16-05-2020 23:00
	//			duration: "6hrs"
	//		upgrade:
	//			window: "3 Jan 1980 - 17 Jan 1980"
	Maintenance RHMIConfigStatusMaintenance `json:"maintenance,omitempty"`
	Upgrade     RHMIConfigStatusUpgrade     `json:"upgrade,omitempty"`
}

type RHMIConfigStatusMaintenance struct {
	ApplyFrom string `json:"applyFrom,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

type RHMIConfigStatusUpgrade struct {
	Window string `json:"window,omitempty"`
}

const DateFormat = "2 Jan 2006 15:04"

type Upgrade struct {
	// contacts: list of contacts which are comma separated
	// "user1@example.com,user2@example.com"
	Contacts string `json:"contacts,omitempty"`
	// always-immediately: boolean value, if set to true an upgrade will be applied as soon as it is available,
	// whether service affecting or not.
	// This takes precedence over all other options
	AlwaysImmediately bool `json:"alwaysImmediately"`
	// during-next-maintenance: boolean value, if set to true an upgrade will be applied within the next maintenance window.
	// Takes precedence over apply-on
	DuringNextMaintenance bool `json:"duringNextMaintenance"`
	// apply-on: string date value. If 'always-immediately' or 'during-next-maintenance' is not set the customer is
	// required to pick a time for the upgrade. Time value will be validated by a webhook and reset to blank after
	// upgrade has completed. Format: "dd MMM YYYY hh:mm" > "12 Jan 1980 23:00". UTC time
	ApplyOn string `json:"applyOn,omitempty"`
}

type Maintenance struct {
	// apply-from: string, day time. Currently this is a 6 hour window.
	// Format: "DDD hh:mm" > "sun 23:00". UTC time
	ApplyFrom string `json:"applyFrom,omitempty"`
}

type Backup struct {
	// apply-on: string, day time.
	// Format: "DDD hh:mm" > "wed 20:00". UTC time
	ApplyOn string `json:"applyOn,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHMIConfig is the Schema for the rhmiconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rhmiconfigs,scope=Namespaced
type RHMIConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RHMIConfigSpec   `json:"spec,omitempty"`
	Status RHMIConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RHMIConfigList contains a list of RHMIConfig
type RHMIConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHMIConfig `json:"items"`
}

func (c *RHMIConfig) ValidateCreate() error {
	return nil
}

func (c *RHMIConfig) ValidateUpdate(old runtime.Object) error {
	if c.Spec.Upgrade.ApplyOn == "" {
		return nil
	}

	if c.Spec.Upgrade.AlwaysImmediately || c.Spec.Upgrade.DuringNextMaintenance {
		return errors.New("spec.Upgrade.ApplyOn shouldn't be set when spec.Upgrade.AlwaysImmediatly or spec.Upgrade.DuringNextMaintenance are true")
	}

	applyOn, err := time.Parse(DateFormat, c.Spec.Upgrade.ApplyOn)
	if err != nil {
		return fmt.Errorf("Invalid value for spec.Upgrade.ApplyOn, must be a date with the format %s", DateFormat)
	}

	if !applyOn.UTC().After(time.Now().UTC()) {
		return fmt.Errorf("Invalid value for spec.Upgrade.ApplyOn: %s. It must be a future date", applyOn.Format(DateFormat))
	}

	return nil
}

func (c *RHMIConfig) ValidateDelete() error {
	return nil
}

// +k8s:deepcopy-gen=false
type rhmiConfigMutatingHandler struct {
	decoder *admission.Decoder
}

func NewRHMIConfigMutatingHandler() admission.Handler {
	return &rhmiConfigMutatingHandler{}
}

func (h *rhmiConfigMutatingHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *rhmiConfigMutatingHandler) Handle(ctx context.Context, request admission.Request) admission.Response {
	rhmiConfig := &RHMIConfig{}
	if err := h.decoder.Decode(request, rhmiConfig); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if rhmiConfig.Annotations == nil {
		rhmiConfig.Annotations = map[string]string{}
	}

	if request.UserInfo.Username != "system:serviceaccount:redhat-rhmi-operator:rhmi-operator" {
		rhmiConfig.Annotations["lastEditUsername"] = request.UserInfo.Username
		rhmiConfig.Annotations["lastEditTimestamp"] = time.Now().UTC().Format(DateFormat)
	}

	marshalled, err := json.Marshal(rhmiConfig)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(request.Object.Raw, marshalled)
}

func init() {
	SchemeBuilder.Register(&RHMIConfig{}, &RHMIConfigList{})
}
