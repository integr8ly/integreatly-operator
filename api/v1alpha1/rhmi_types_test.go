package v1alpha1

import (
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRHMI_IsCoreComponentsHealthy(t *testing.T) {
	statusFactory := func(phase1, phase2, phase3 StatusPhase) RHMIStatus {
		return RHMIStatus{Stages: map[StageName]RHMIStageStatus{
			InstallStage: {
				Name: InstallStage,
				Products: map[ProductName]RHMIProductStatus{
					Product3Scale: {
						Phase: phase1,
					},
					ProductRHSSOUser: {
						Phase: phase2,
					},
					ProductCloudResources: {
						Phase: phase3,
					},
				},
			},
		}}
	}

	type fields struct {
		Status RHMIStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "test true if all core components in phase complete",
			fields: fields{Status: statusFactory(PhaseCompleted, PhaseCompleted, PhaseCompleted)},
			want:   true,
		},
		{
			name:   "test false if 3scale component not in phase complete",
			fields: fields{Status: statusFactory(PhaseFailed, PhaseCompleted, PhaseCompleted)},
			want:   false,
		},
		{
			name:   "test false if User SSO component not in phase complete",
			fields: fields{Status: statusFactory(PhaseCompleted, PhaseFailed, PhaseCompleted)},
			want:   false,
		},
		{
			name:   "test false if Cloud Resource component not in phase complete",
			fields: fields{Status: statusFactory(PhaseCompleted, PhaseCompleted, PhaseFailed)},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				Status: tt.fields.Status,
			}
			if got := i.IsCoreComponentsHealthy(); got != tt.want {
				t.Errorf("IsCoreComponentsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMI_IsDegraded(t *testing.T) {
	type fields struct {
		TypeMeta   v1.TypeMeta
		ObjectMeta v1.ObjectMeta
		Spec       RHMISpec
		Status     RHMIStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "test true if stage not in phase complete and not uninstalling ",
			fields: fields{Status: RHMIStatus{Stage: BootstrapStage}},
			want:   true,
		},
		{
			name:   "test false if stage in phase complete and not uninstalling",
			fields: fields{Status: RHMIStatus{Stage: CompleteStage}},
			want:   false,
		},
		{
			name:   "test false if stage in phase complete and uninstalling",
			fields: fields{ObjectMeta: v1.ObjectMeta{DeletionTimestamp: &v1.Time{Time: time.Now()}}, Status: RHMIStatus{Stage: CompleteStage}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := i.IsDegraded(); got != tt.want {
				t.Errorf("IsDegraded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMI_IsInstallBlocked(t *testing.T) {
	type fields struct {
		ObjectMeta v1.ObjectMeta
		Status     RHMIStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "test true when install is over 2 hours old and version is not set",
			fields: fields{ObjectMeta: v1.ObjectMeta{CreationTimestamp: v1.Time{Time: time.Now().Add(-2 * time.Hour)}}, Status: RHMIStatus{Version: ""}},
			want:   true,
		},
		{
			name:   "test false when install is over 2 hours old and version is set",
			fields: fields{ObjectMeta: v1.ObjectMeta{CreationTimestamp: v1.Time{Time: time.Now().Add(-2 * time.Hour)}}, Status: RHMIStatus{Version: "0.0.0"}},
			want:   false,
		},
		{
			name:   "test false when install is under 2 hours old and version is set",
			fields: fields{ObjectMeta: v1.ObjectMeta{CreationTimestamp: v1.Time{Time: time.Now().Add(-60 * time.Minute)}}, Status: RHMIStatus{Version: "0.0.0"}},
			want:   false,
		},
		{
			name:   "test false when install is under 2 hours old and version is set and uninstalling",
			fields: fields{ObjectMeta: v1.ObjectMeta{CreationTimestamp: v1.Time{Time: time.Now().Add(-60 * time.Minute)}, DeletionTimestamp: &v1.Time{Time: time.Now()}}, Status: RHMIStatus{Version: "0.0.0"}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				ObjectMeta: tt.fields.ObjectMeta,
				Status:     tt.fields.Status,
			}
			if got := i.IsInstallBlocked(); got != tt.want {
				t.Errorf("IsInstallBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMI_IsInstalled(t *testing.T) {
	type fields struct {
		TypeMeta   v1.TypeMeta
		ObjectMeta v1.ObjectMeta
		Spec       RHMISpec
		Status     RHMIStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "test true when version is not empty",
			fields: fields{Status: RHMIStatus{
				Version: "0.0.0",
			}},
			want: true,
		},
		{
			name: "test false when version is empty",
			fields: fields{Status: RHMIStatus{
				Version: "",
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := i.IsInstalled(); got != tt.want {
				t.Errorf("IsInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMI_IsProductPhaseComplete(t *testing.T) {
	productName := ProductCloudResources
	statusFactory := func(phase StatusPhase) RHMIStatus {
		return RHMIStatus{
			Stages: map[StageName]RHMIStageStatus{
				InstallStage: {
					Name: InstallStage,
					Products: map[ProductName]RHMIProductStatus{
						productName: {
							Phase: phase,
						},
					},
				},
			},
		}
	}

	type fields struct {
		Status RHMIStatus
	}
	type args struct {
		productName ProductName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test true when product is phase complete",
			fields: fields{Status: statusFactory(PhaseCompleted)},
			args:   args{productName: productName},
			want:   true,
		},
		{
			name:   "test false when product is not phase complete",
			fields: fields{Status: statusFactory(PhaseAwaitingComponents)},
			args:   args{productName: productName},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				Status: tt.fields.Status,
			}
			if got := i.IsProductInInstallStagePhaseComplete(tt.args.productName); got != tt.want {
				t.Errorf("IsProductPhaseComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMI_IsUninstallBlocked(t *testing.T) {
	type fields struct {
		ObjectMeta v1.ObjectMeta
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "test uninstall is blocked if deletion timestamp is over 2 hours old",
			fields: fields{ObjectMeta: v1.ObjectMeta{DeletionTimestamp: &v1.Time{Time: time.Now().Add(-2 * time.Hour)}}},
			want:   true,
		},
		{
			name:   "test uninstall is not blocked if deletion timestamp is under 2 hours old",
			fields: fields{ObjectMeta: v1.ObjectMeta{DeletionTimestamp: &v1.Time{Time: time.Now().Add(-119 * time.Minute)}}},
			want:   false,
		},
		{
			name:   "test uninstall is not blocked if not uninstalling",
			fields: fields{ObjectMeta: v1.ObjectMeta{DeletionTimestamp: nil}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				ObjectMeta: tt.fields.ObjectMeta,
			}
			if got := i.IsUninstallBlocked(); got != tt.want {
				t.Errorf("IsUninstallBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMI_IsUninstalling(t *testing.T) {
	type fields struct {
		ObjectMeta v1.ObjectMeta
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "test true when deletion timestamp is not nil",
			fields: fields{ObjectMeta: v1.ObjectMeta{DeletionTimestamp: &v1.Time{Time: time.Now()}}},
			want:   true,
		},
		{
			name:   "test false when deletion timestamp is nil",
			fields: fields{ObjectMeta: v1.ObjectMeta{DeletionTimestamp: nil}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &RHMI{
				ObjectMeta: tt.fields.ObjectMeta,
			}
			if got := i.IsUninstalling(); got != tt.want {
				t.Errorf("IsUninstalling() = %v, want %v", got, tt.want)
			}
		})
	}
}
