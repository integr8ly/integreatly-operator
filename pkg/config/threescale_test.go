package config

import (
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
)

func TestThreeScale_GetBackendRedisNodeSize(t1 *testing.T) {
	type fields struct {
		config ProductConfig
	}
	type args struct {
		activeQuota string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "test empty string is returned when active quota is not 100 M",
			args: args{
				activeQuota: quota.OneHundredThousandQuotaName,
			},
			want: "",
		},
		{
			name: "test cache.m5.large is returned when active quota is 100 M",
			args: args{
				activeQuota: quota.OneHundredMillionQuotaName,
			},
			want: "cache.m5.large",
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &ThreeScale{
				config: tt.fields.config,
			}
			if got := t.GetBackendRedisNodeSize(tt.args.activeQuota); got != tt.want {
				t1.Errorf("GetBackendRedisNodeSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
