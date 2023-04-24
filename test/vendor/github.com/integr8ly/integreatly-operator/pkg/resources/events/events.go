package events

import (
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"k8s.io/client-go/tools/record"
)

// Emits a normal event upon successful completion of stage reconcile
func HandleStageComplete(recorder record.EventRecorder, installation *integreatlyv1alpha1.RHMI, stageName integreatlyv1alpha1.StageName) {
	stageStatus := installation.Status.Stages[stageName]
	if stageStatus.Phase != integreatlyv1alpha1.PhaseCompleted {
		recorder.Event(installation, "Normal", integreatlyv1alpha1.EventInstallationCompleted, fmt.Sprintf("%s stage has reconciled successfully", stageName))
	}
}

// Emits a normal event upon successful completion of product installation
func HandleProductComplete(recorder record.EventRecorder, installation *integreatlyv1alpha1.RHMI, stageName integreatlyv1alpha1.StageName, productName integreatlyv1alpha1.ProductName) {
	stage := installation.Status.Stages[stageName]
	if stage.Products[productName].Phase != integreatlyv1alpha1.PhaseCompleted {
		recorder.Event(installation, "Normal", integreatlyv1alpha1.EventInstallationCompleted, fmt.Sprintf("%s was installed successfully", productName))
	}
}

// Emits a warning event when a processing error occurs during reconcile. It is only emitted on phase failed
func HandleError(recorder record.EventRecorder, installation *integreatlyv1alpha1.RHMI, phase integreatlyv1alpha1.StatusPhase, errorMessage string, err error) {
	if err != nil && phase == integreatlyv1alpha1.PhaseFailed {
		recorder.Event(installation, "Warning", integreatlyv1alpha1.EventProcessingError, fmt.Sprintf("%s:\n%s", errorMessage, err.Error()))
	}
}
