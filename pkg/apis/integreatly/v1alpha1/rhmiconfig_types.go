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
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	DefaultBackupApplyOn        = "03:01"
	DefaultMaintenanceApplyFrom = "Thu 02:00"
)

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

	// Scheduled contains the information on the next upgrade schedule
	Scheduled *UpgradeSchedule `json:"scheduled,omitempty"`
}

type UpgradeSchedule struct {
	// For is the calculated time when the upgrade is scheduled for, in format "2 Jan 2006 15:04"
	For string `json:"for,omitempty"`

	// CalculatedFrom shows how the "For" value is calculated. The following
	// values are possible:
	// * ApplyOn: When a value for "spec.Upgrade.ApplyOn" is set, the upgrade is
	//   scheduled for that date
	// * NextMaintenance: When "spec.Upgrade.DuringNextMaintenance" is true, the
	//   upgrade is scheduled for the next maintenance date,
	//   calculated from "spec.Maintenance"
	// * TwoWeeksMaintenanceWindow: If no value is set for "spec.Upgrade", the
	//   upgrade is scheduled for the next maintenance window two weeks after
	//   the install plan is created
	// * DefaultTwoWeeks: If no value is set for "spec.Upgrade" and no maintenance
	//   window has been specified in "spec.Maintenance", the default schedule
	//   is two weeks after the install plan is created
	CalculatedFrom UpgradeScheduleCalculation `json:"calculatedFrom,omitempty"`
}

type UpgradeScheduleCalculation string

const DefaultTwoWeeks UpgradeScheduleCalculation = "DefaultTwoWeeks"
const TwoWeeksMaintenanceWindow UpgradeScheduleCalculation = "TwoWeeksMaintenanceWindow"
const NextMaintenance UpgradeScheduleCalculation = "NextMaintenance"
const ApplyOn UpgradeScheduleCalculation = "ApplyOn"

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
	if _, _, err := ValidateBackupAndMaintenance(c.Spec.Backup.ApplyOn, c.Spec.Maintenance.ApplyFrom); err != nil {
		return err
	}

	if c.Spec.Upgrade.ApplyOn == "" {
		return nil
	}

	if c.Spec.Upgrade.AlwaysImmediately || c.Spec.Upgrade.DuringNextMaintenance {
		return errors.New("spec.Upgrade.ApplyOn shouldn't be set when spec.Upgrade.AlwaysImmediatly or spec.Upgrade.DuringNextMaintenance are true")
	}

	applyOn, err := time.Parse(DateFormat, c.Spec.Upgrade.ApplyOn)
	if err != nil {
		return fmt.Errorf("invalid value for spec.Upgrade.ApplyOn, must be a date with the format %s", DateFormat)
	}

	if !applyOn.UTC().After(time.Now().UTC()) {
		return fmt.Errorf("invalid value for spec.Upgrade.ApplyOn: %s. It must be a future date", applyOn.Format(DateFormat))
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
	
	if rhmiConfig.Spec.Maintenance.ApplyFrom == "" {
		rhmiConfig.Spec.Maintenance.ApplyFrom = DefaultMaintenanceApplyFrom
	}
	if rhmiConfig.Spec.Backup.ApplyOn == "" {
		rhmiConfig.Spec.Backup.ApplyOn = DefaultBackupApplyOn
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

// ValidateBackupAndMaintenance ensures that the backup and maintenance times
//   * are not empty
//   * are correctly formatted
//   * do not overlap
// If the times are valid, 1 hour non-overlapping backup and maintenance windows
// are returned as a result in a format required by AWS
func ValidateBackupAndMaintenance(backupApplyOn, maintenanceApplyFrom string) (string, string, error) {
	// we accept a blank string for both ApplyOn and ApplyFrom
	// in the case where these values are empty the RHMIConfig controller will set them to the expected defaults
	// despite defaults we need to still validate to catch cases where one value is a blank string
	// we need to ensure there are no overlapping time values
	if maintenanceApplyFrom == "" {
		maintenanceApplyFrom = DefaultMaintenanceApplyFrom
	}
	if backupApplyOn == "" {
		backupApplyOn = DefaultBackupApplyOn
	}

	// ensure backup applyOn format is correct
	parsedBackupTime, err := time.Parse("15:04", backupApplyOn)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse backup ApplyOn value : expected format HH:mm : %v", err)
	}

	// ensure maintenance applyFrom format is correct
	// we expect a format of: `DDD HH:mm`
	maintenanceSegments := strings.Split(maintenanceApplyFrom, " ")
	if len(maintenanceSegments) != 2 {
		return "", "", fmt.Errorf("failed to parse maintenance ApplyFrom value : expected format DDD HH:mm , found format %s", maintenanceApplyFrom)
	}
	maintenanceDay := maintenanceSegments[0]
	maintenanceTime := maintenanceSegments[1]

	// verify maintenance day is valid
	var expectedDays = []string{
		"sun",
		"mon",
		"tue",
		"wed",
		"thu",
		"fri",
		"sat",
	}
	if !contains(expectedDays, strings.ToLower(maintenanceDay)) {
		return "", "", fmt.Errorf("formatting failure, found invalid maintenance applyFrom value. Expected: `DDD HH:mm` found: %s", maintenanceApplyFrom)
	}

	// verify maintenance time is valid
	parsedMaintenanceTime, err := time.Parse("15:04", maintenanceTime)
	if err != nil {
		return "", "", fmt.Errorf("failure while parsing maintenance applyFrom value. Format expected: `DDD HH:mm` found: %s: %v", maintenanceApplyFrom, err)
	}

	// add an hour to maintenace and backup time to create the windows.
	// we require a minimum of 1hr windows for both maintenance and backup
	// these windows cannot overlap, this is a requirement of AWS
	parsedMaintenanceTimePlusOneHour := parsedMaintenanceTime.Add(time.Hour)
	parsedBackupTimePlusOneHour := parsedBackupTime.Add(time.Hour)

	// build expected maintenance window strings for error message,
	builtMaintenanceString := fmt.Sprintf("%02d:%02d-%02d:%02d", parsedMaintenanceTime.Hour(), parsedMaintenanceTime.Minute(), parsedMaintenanceTimePlusOneHour.Hour(), parsedMaintenanceTimePlusOneHour.Minute())
	builtBackupString := fmt.Sprintf("%02d:%02d-%02d:%02d", parsedBackupTime.Hour(), parsedBackupTime.Minute(), parsedBackupTimePlusOneHour.Hour(), parsedBackupTimePlusOneHour.Minute())

	// ensure backup and maintenance time ranges do not overlap
	// we expect RHMI operator to validate the ranges, as a sanity check we perform an extra validation here
	// this is to avoid an obscure error message from AWS when we apply the times
	// http://baodad.blogspot.com/2014/06/date-range-overlap.html
	// (StartA <= EndB)  and  (EndA >= StartB)
	if timeBlockOverlaps(parsedBackupTime, parsedBackupTimePlusOneHour, parsedMaintenanceTime, parsedMaintenanceTimePlusOneHour) {
		return "", "", fmt.Errorf("backup and maintenance times cannot overlap, each time is parsed as a 1 hour window, current backup applyOn window : %s overlaps with current maintenance window : %s ", builtBackupString, builtMaintenanceString)
	}
	return backupApplyOn, maintenanceApplyFrom, nil
}

// timeBlockOverlaps checks if two time ranges overlap and returns true
// if they do
func timeBlockOverlaps(startA, endA, startB, endB time.Time) bool {
	return startA.Unix() <= endB.Unix() && endA.Unix() >= startB.Unix()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
