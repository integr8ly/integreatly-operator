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
	"fmt"
	"path"

	addonv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RHMIConditionType string

func (c RHMIConditionType) String() string {
	return path.Join(group, string(c))
}

const HealthyConditionType RHMIConditionType = "Healthy"

func (i *RHMI) InstalledCondition() metav1.Condition {
	return addoninstance.NewAddonInstanceConditionInstalled(
		metav1.ConditionTrue,
		addonv1alpha1.AddonInstanceInstalledReasonSetupComplete,
		"Installation complete",
	)
}

func (i *RHMI) InstallBlockedCondition() metav1.Condition {
	return addoninstance.NewAddonInstanceConditionInstalled(
		metav1.ConditionFalse,
		addonv1alpha1.AddonInstanceInstalledReasonBlocked,
		"Installation blocked",
	)
}

func (i *RHMI) UninstalledCondition() metav1.Condition {
	return addoninstance.NewAddonInstanceConditionInstalled(
		metav1.ConditionFalse,
		addonv1alpha1.AddonInstanceInstalledReasonTeardownComplete,
		"Teardown complete",
	)
}

func (i *RHMI) UninstallBlockedCondition() metav1.Condition {
	return addoninstance.NewAddonInstanceConditionInstalled(
		metav1.ConditionTrue,
		addonv1alpha1.AddonInstanceInstalledReasonBlocked,
		"Teardown blocked",
	)
}

func (i *RHMI) HealthyCondition() metav1.Condition {
	return newRHMICondition(HealthyConditionType, metav1.ConditionTrue, "Healthy", "Core components healthy")
}

func (i *RHMI) UnHealthyCondition() metav1.Condition {
	return newRHMICondition(HealthyConditionType, metav1.ConditionFalse, "Healthy", "One or more core components unhealthy")
}

func (i *RHMI) DegradedCondition() metav1.Condition {
	return addoninstance.NewAddonInstanceConditionDegraded(
		metav1.ConditionTrue,
		string(addonv1alpha1.AddonInstanceConditionDegraded),
		fmt.Sprintf("Components degraded: %s", i.GetDegradedComponents()),
	)
}

func (i *RHMI) NonDegradedCondition() metav1.Condition {
	return addoninstance.NewAddonInstanceConditionDegraded(
		metav1.ConditionFalse,
		string(addonv1alpha1.AddonInstanceConditionDegraded),
		"All components healthy",
	)
}

func (i *RHMI) ReadyToBeDeletedCondition() metav1.Condition {
	return metav1.Condition{
		Type:    addonv1alpha1.AddonInstanceConditionReadyToBeDeleted.String(),
		Status:  metav1.ConditionTrue,
		Reason:  string(addonv1alpha1.AddonInstanceReasonReadyToBeDeleted),
		Message: "Teardown complete",
	}
}

func newRHMICondition(conditionType RHMIConditionType, conditionStatus metav1.ConditionStatus, reason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionType.String(),
		Status:  conditionStatus,
		Reason:  reason,
		Message: msg,
	}
}
