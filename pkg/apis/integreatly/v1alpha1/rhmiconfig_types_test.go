package v1alpha1

import "testing"

func TestValidateBackupAndMaintenance(t *testing.T) {
	type args struct {
		backupApplyOn        string
		maintenanceApplyFrom string
	}
	tests := []struct {
		name            string
		args            args
		wantBackup      string
		wantMaintenance string
		wantErr         bool
	}{
		{
			name: "test exact overlapping window fails",
			args: args{
				backupApplyOn:        "01:00",
				maintenanceApplyFrom: "mon 02:00",
			},
			wantErr: true,
		},
		{
			name: "test overlapping window fails",
			args: args{
				backupApplyOn:        "01:30",
				maintenanceApplyFrom: "mon 01:00",
			},
			wantErr: true,
		},
		{
			name: "test non-overlapping window succeeds with backup before maintenance",
			args: args{
				backupApplyOn:        "01:00",
				maintenanceApplyFrom: "mon 02:01",
			},
			wantBackup:      "01:00",
			wantMaintenance: "mon 02:01",
		},
		{
			name: "test non-overlapping window succeeds with backup after maintenance",
			args: args{
				backupApplyOn:        "01:31",
				maintenanceApplyFrom: "mon 00:30",
			},
			wantBackup:      "01:31",
			wantMaintenance: "mon 00:30",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBackup, gotMaintenance, err := ValidateBackupAndMaintenance(tt.args.backupApplyOn, tt.args.maintenanceApplyFrom)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBackupAndMaintenance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotBackup != tt.wantBackup {
				t.Errorf("ValidateBackupAndMaintenance() got = %v, want %v", gotBackup, tt.wantBackup)
			}
			if gotMaintenance != tt.wantMaintenance {
				t.Errorf("ValidateBackupAndMaintenance() got1 = %v, want %v", gotMaintenance, tt.wantMaintenance)
			}
		})
	}
}
