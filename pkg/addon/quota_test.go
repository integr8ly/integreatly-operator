package addon

import (
	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"strings"
	"testing"
)

func TestGetQuotaConfig(t *testing.T) {
	type args struct {
		installType string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get quota config for rhoam multi tenant",
			args: args{
				installType: string(v1alpha1.InstallationTypeMultitenantManagedApi),
			},
			want: strings.TrimSpace(mtQuotaConfig),
		},
		{
			name: "get quota config for all other types",
			args: args{
				installType: string(v1alpha1.InstallationTypeManagedApi),
			},
			want: strings.TrimSpace(quotaConfig),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetQuotaConfig(tt.args.installType); got != tt.want {
				t.Errorf("GetQuotaConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
