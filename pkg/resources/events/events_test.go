package events

import (
	"errors"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/client-go/tools/record"
)

const (
	stageName   = "stageName"
	productName = "testProduct"
)

type EventsScenario struct {
	Name               string
	Installation       *integreatlyv1alpha1.RHMI
	ExpectedEventCount int
	Error              error
	ErrorMessage       string
	StatusPhase        integreatlyv1alpha1.StatusPhase
}

func TestHandleStageComplete(t *testing.T) {
	cases := []EventsScenario{
		{
			Name:               "test stage complete event handler on a stage thats unavailable",
			Installation:       &integreatlyv1alpha1.RHMI{},
			ExpectedEventCount: 1,
		},
		{
			Name: "test stage complete event handler on a stage thats not completed",
			Installation: &integreatlyv1alpha1.RHMI{
				Status: integreatlyv1alpha1.RHMIStatus{
					Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
						stageName: integreatlyv1alpha1.RHMIStageStatus{
							Name:  stageName,
							Phase: integreatlyv1alpha1.PhaseInProgress,
						},
					},
				},
			},
			ExpectedEventCount: 1,
		},
		{
			Name: "test stage complete event handler on a stage thats completed",
			Installation: &integreatlyv1alpha1.RHMI{
				Status: integreatlyv1alpha1.RHMIStatus{
					Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
						stageName: integreatlyv1alpha1.RHMIStageStatus{
							Name:  stageName,
							Phase: integreatlyv1alpha1.PhaseCompleted,
						},
					},
				},
			},
			ExpectedEventCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			recorder := record.NewFakeRecorder(1)
			HandleStageComplete(recorder, tc.Installation, stageName)

			if len(recorder.Events) != tc.ExpectedEventCount {
				t.Fatalf("Expected event count %d but got %d", tc.ExpectedEventCount, len(recorder.Events))
			}
		})
	}
}

func TestHandleProductComplete(t *testing.T) {
	cases := []EventsScenario{
		{
			Name:               "test product complete event handler on a stage thats unavailable",
			Installation:       &integreatlyv1alpha1.RHMI{},
			ExpectedEventCount: 1,
		},
		{
			Name: "test product complete event handler on a product thats unavailable",
			Installation: &integreatlyv1alpha1.RHMI{
				Status: integreatlyv1alpha1.RHMIStatus{
					Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
						stageName: integreatlyv1alpha1.RHMIStageStatus{
							Name:     stageName,
							Phase:    integreatlyv1alpha1.PhaseInProgress,
							Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{},
						},
					},
				},
			},
			ExpectedEventCount: 1,
		},
		{
			Name: "test product complete event handler on a product thats in progress",
			Installation: &integreatlyv1alpha1.RHMI{
				Status: integreatlyv1alpha1.RHMIStatus{
					Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
						stageName: integreatlyv1alpha1.RHMIStageStatus{
							Name:  stageName,
							Phase: integreatlyv1alpha1.PhaseInProgress,
							Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
								productName: integreatlyv1alpha1.RHMIProductStatus{
									Name:   productName,
									Status: integreatlyv1alpha1.PhaseInProgress,
								},
							},
						},
					},
				},
			},
			ExpectedEventCount: 1,
		},
		{
			Name: "test product complete event handler on a product thats completed",
			Installation: &integreatlyv1alpha1.RHMI{
				Status: integreatlyv1alpha1.RHMIStatus{
					Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
						stageName: integreatlyv1alpha1.RHMIStageStatus{
							Name:  stageName,
							Phase: integreatlyv1alpha1.PhaseInProgress,
							Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
								productName: integreatlyv1alpha1.RHMIProductStatus{
									Name:   productName,
									Status: integreatlyv1alpha1.PhaseCompleted,
								},
							},
						},
					},
				},
			},
			ExpectedEventCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			recorder := record.NewFakeRecorder(1)
			HandleProductComplete(recorder, tc.Installation, stageName, productName)

			if len(recorder.Events) != tc.ExpectedEventCount {
				t.Fatalf("Expected event count %d but got %d", tc.ExpectedEventCount, len(recorder.Events))
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	cases := []EventsScenario{
		{
			Name:               "test error event handler with no errors and phase not failed",
			Installation:       &integreatlyv1alpha1.RHMI{},
			ErrorMessage:       "failed installation",
			ExpectedEventCount: 0,
		},
		{
			Name:               "test error event handler with an error and phase not failed",
			Installation:       &integreatlyv1alpha1.RHMI{},
			ExpectedEventCount: 0,
			Error:              errors.New("an error occurred"),
			ErrorMessage:       "failed installation",
		},
		{
			Name:               "test error event handler with no errors and phase failed",
			Installation:       &integreatlyv1alpha1.RHMI{},
			ExpectedEventCount: 0,
			StatusPhase:        integreatlyv1alpha1.PhaseFailed,
			ErrorMessage:       "failed installation",
		},
		{
			Name:               "test error event handler with an error and phase failed",
			Installation:       &integreatlyv1alpha1.RHMI{},
			ExpectedEventCount: 1,
			Error:              errors.New("an error occurred"),
			ErrorMessage:       "failed installation",
			StatusPhase:        integreatlyv1alpha1.PhaseFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			recorder := record.NewFakeRecorder(1)
			HandleError(recorder, tc.Installation, tc.StatusPhase, tc.ErrorMessage, tc.Error)

			if len(recorder.Events) != tc.ExpectedEventCount {
				t.Fatalf("Expected event count %d but got %d", tc.ExpectedEventCount, len(recorder.Events))
			}
		})
	}
}
