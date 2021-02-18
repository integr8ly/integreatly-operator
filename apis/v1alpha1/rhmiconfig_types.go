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

package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	DefaultBackupApplyOn        = "03:01"
	DefaultMaintenanceApplyFrom = "Thu 02:00"

	DefaultNotBeforeDays      = 7
	DefaultWaitForMaintenance = true

	// Maximum allowed number of days to schedule an upgrade via `NotBeforeDays`
	// MaxUpgradeDays = 14
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
	Maintenance      RHMIConfigStatusMaintenance `json:"maintenance,omitempty"`
	Upgrade          RHMIConfigStatusUpgrade     `json:"upgrade,omitempty"`
	UpgradeAvailable *UpgradeAvailable           `json:"upgradeAvailable,omitempty"`
}

type RHMIConfigStatusMaintenance struct {
	ApplyFrom string `json:"applyFrom,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

type RHMIConfigStatusUpgrade struct {
	// Scheduled contains the information on the next upgrade schedule
	Scheduled *UpgradeSchedule `json:"scheduled,omitempty"`
}

type UpgradeSchedule struct {
	// For is the calculated time when the upgrade is scheduled for, in format "2 Jan 2006 15:04"
	For string `json:"for,omitempty"`
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

	// If this value is true, upgrades will be approved in the next maintenance window
	// n days after the upgrade is made available. Being n the value of `notBeforeDays`.
	// +optional
	// +nullable
	WaitForMaintenance *bool `json:"waitForMaintenance,omitempty"`

	// Minimum of days since an upgrade is made available until it's approved
	// +optional
	// +nullable
	NotBeforeDays *int `json:"notBeforeDays,omitempty"`

	Schedule *bool `json:"schedule,omitempty"`
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

type UpgradeAvailable struct {
	// Time of new update becoming available
	// Format: "DDD hh:mm" > "sun 23:00". UTC time
	AvailableAt v1.Time `json:"availableAt,omitempty" protobuf:"bytes,8,opt,name=availableAt"`

	// target-version: string, version of incoming RHMI Operator
	TargetVersion string `json:"targetVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RHMIConfig is the Schema for the rhmiconfigs API
type RHMIConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RHMIConfigSpec   `json:"spec,omitempty"`
	Status RHMIConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RHMIConfigList contains a list of RHMIConfig
type RHMIConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHMIConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RHMIConfig{}, &RHMIConfigList{})
}

func (c *RHMIConfig) ValidateCreate() error {
	return nil
}

func (c *RHMIConfig) ValidateUpdate(old runtime.Object) error {
	if _, _, err := ValidateBackupAndMaintenance(c.Spec.Backup.ApplyOn, c.Spec.Maintenance.ApplyFrom); err != nil {
		return err
	}

	// Validate the NotBeforeDays. Must be an integer n where
	// n > 0 && n <= MaxUpgradeDays
	if c.Spec.Upgrade.NotBeforeDays != nil {
		notBeforeDays := *c.Spec.Upgrade.NotBeforeDays

		if notBeforeDays < 0 {
			return errors.New("Value of spec.Upgrade.NotBeforeDays must be greater or equal to zero")
		}
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

	saName := "system:serviceaccount:rhmi-operator"
	if request.UserInfo.Username != saName {
		rhmiConfig.Annotations["lastEditUsername"] = request.UserInfo.Username
		rhmiConfig.Annotations["lastEditTimestamp"] = time.Now().UTC().Format(DateFormat)
	}

	if rhmiConfig.Spec.Maintenance.ApplyFrom == "" {
		rhmiConfig.Spec.Maintenance.ApplyFrom = DefaultMaintenanceApplyFrom
	}
	if rhmiConfig.Spec.Backup.ApplyOn == "" {
		rhmiConfig.Spec.Backup.ApplyOn = DefaultBackupApplyOn
	}

	defaultNotBeforeDays := DefaultNotBeforeDays
	defaultWaitForMaintenance := DefaultWaitForMaintenance

	defaultUpgradeSpec := &Upgrade{
		NotBeforeDays:      &defaultNotBeforeDays,
		WaitForMaintenance: &defaultWaitForMaintenance,
	}

	oldUpgradeSpec := &Upgrade{}
	oldRhmiConfig := &RHMIConfig{}
	if err := h.decoder.DecodeRaw(request.OldObject, oldRhmiConfig); err == nil {
		oldUpgradeSpec = &oldRhmiConfig.Spec.Upgrade
	}

	rhmiConfig.Spec.Upgrade.WaitForMaintenance = either(
		rhmiConfig.Spec.Upgrade.WaitForMaintenance,
		oldUpgradeSpec.WaitForMaintenance,
		defaultUpgradeSpec.WaitForMaintenance,
	).(*bool)

	rhmiConfig.Spec.Upgrade.NotBeforeDays = either(
		rhmiConfig.Spec.Upgrade.NotBeforeDays,
		oldUpgradeSpec.NotBeforeDays,
		defaultUpgradeSpec.NotBeforeDays,
	).(*int)

	marshalled, err := json.Marshal(rhmiConfig)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(request.Object.Raw, marshalled)
}

func (u *Upgrade) DefaultIfEmpty() {
	u.NotBeforeDays = either(u.NotBeforeDays, DefaultNotBeforeDays).(*int)
	u.WaitForMaintenance = either(u.WaitForMaintenance, DefaultWaitForMaintenance).(*bool)
	u.Schedule = either(u.Schedule, false).(*bool)
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

// Either takes a list of elements of type a or *a, and returns a pointer to the
// first element that is either not a pointer or, if it's a pointer, is not nil
func either(values ...interface{}) interface{} {
	for _, value := range values {
		refValue := reflect.ValueOf(value)

		if refValue.Kind() != reflect.Ptr {
			res := reflect.New(reflect.TypeOf(value))
			res.Elem().Set(refValue)
			return res.Interface()
		}

		if !refValue.IsNil() {
			return value
		}
	}

	return nil
}
