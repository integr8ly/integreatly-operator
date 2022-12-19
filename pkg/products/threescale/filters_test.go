package threescale

import (
	v1 "github.com/openshift/api/route/v1"
	"reflect"
	"testing"
)

func TestNewTenantAccountsFilter(t *testing.T) {
	type args struct {
		accounts []AccountDetail
	}
	tests := []struct {
		name string
		args args
		want TenantAccountsFilter
	}{
		{
			name: "Accounts is nil",
			args: args{accounts: nil},
			want: TenantAccountsFilter{providers: nil, developers: nil},
		},
		{
			name: "Accounts have 3 routes of which 2 are admin",
			args: args{accounts: []AccountDetail{
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "test1-example.com",
				},
				{
					Id:           1,
					State:        "approved",
					AdminBaseURL: "test2-admin.example.com",
				},
				{
					Id:           2,
					State:        "approved",
					AdminBaseURL: "test3-admin.example.com",
				},
			}},
			want: TenantAccountsFilter{
				providers: []AccountDetail{
					{
						Id:           0,
						State:        "approved",
						AdminBaseURL: "test1-example.com",
					},
					{
						Id:           1,
						State:        "approved",
						AdminBaseURL: "test2-admin.example.com",
					},
					{
						Id:           2,
						State:        "approved",
						AdminBaseURL: "test3-admin.example.com",
					},
				},
				developers: []developerRoute{
					{
						Url:   "test2.example.com",
						State: "approved",
					},
					{
						Url:   "test3.example.com",
						State: "approved",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTenantAccountsFilter(tt.args.accounts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTenantAccountsFilter()\nGot: %v \nWant %v", got, tt.want)
			}
		})
	}
}

func TestTenantAccountsFilter_SetAccounts(t *testing.T) {

	type args struct {
		accounts []AccountDetail
	}
	tests := []struct {
		name string
		args args
		want []AccountDetail
	}{
		{
			name: "AccountDetail assigned to inner providers correctly",
			args: args{accounts: []AccountDetail{
				{
					Id:   0,
					Name: "Mock 01",
				},
			}},
			want: []AccountDetail{
				{
					Id:   0,
					Name: "Mock 01",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TenantAccountsFilter{}
			f.setProviders(tt.args.accounts)

			if !reflect.DeepEqual(tt.want, f.providers) {
				t.Errorf("Returned Account Details does not macth input. \nWant: %v \nGot: %v", tt.want, f.providers)
			}

		})
	}
}

func TestTenantAccountsFilter_GenerateDeveloperAccounts(t *testing.T) {
	type args struct {
		accounts []AccountDetail
	}
	tests := []struct {
		name string
		args args
		want []developerRoute
	}{
		{
			name: "All accounts are admin",
			args: args{accounts: []AccountDetail{
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "3scale-admin.example.com",
				},
				{
					Id:           1,
					State:        "approved",
					AdminBaseURL: "test-admin.example.com",
				},
			}},
			want: []developerRoute{
				{
					Url:   "3scale.example.com",
					State: "approved",
				},
				{
					Url:   "test.example.com",
					State: "approved",
				},
			},
		},
		{
			name: "One out of two accounts are admin",
			args: args{accounts: []AccountDetail{
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "3scale-admin.example.com",
				},
				{
					Id:           1,
					State:        "approved",
					AdminBaseURL: "test.example.com",
				},
			}},
			want: []developerRoute{
				{
					Url:   "3scale.example.com",
					State: "approved",
				},
			},
		},
		{
			name: "Multiply admin sections in URL",
			args: args{accounts: []AccountDetail{
				{
					Id:           1,
					AdminBaseURL: "x-admin.dev-admin.example.com",
				},
			}},
			want: []developerRoute{
				{
					Url: "x-admin.dev.example.com",
				},
			},
		},
		{
			name: "Input list has no admin urls",
			args: args{accounts: []AccountDetail{
				{
					Id:           1,
					State:        "approved",
					AdminBaseURL: "test.example.com",
				},
			}},
			want: nil,
		},
		{
			name: "Input list is nil",
			args: args{accounts: nil},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TenantAccountsFilter{}
			f.generateDeveloperRoutes(tt.args.accounts)

			if !reflect.DeepEqual(tt.want, f.developers) {
				t.Errorf("developers list not as expected. \nWant: %s \nGot: %s", tt.want, f.developers)
			}

		})
	}
}

func TestTenantAccountsFilter_Developer(t *testing.T) {
	type fields struct {
		developers []developerRoute
	}
	type args struct {
		r v1.Route
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Url exist and is approved",
			fields: fields{developers: []developerRoute{
				{
					Url:   "dummy.example.com",
					State: "approved",
				},
				{
					Url:   "3scale.example.com",
					State: "approved",
				},
			}},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale.example.com",
					},
				},
			},
			want: true,
		},
		{
			name: "Url exist but not approved",
			fields: fields{developers: []developerRoute{
				{
					Url:   "dummy.example.com",
					State: "approved",
				},
				{
					Url:   "3scale.example.com",
					State: "not approved",
				},
			}},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale.example.com",
					},
				},
			},
			want: false,
		},
		{
			name: "Url does not exist",
			fields: fields{developers: []developerRoute{
				{
					Url:   "dummy.example.com",
					State: "approved",
				},
				{
					Url:   "dummy2.example.com",
					State: "approved",
				},
			}},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale.example.com",
					},
				},
			},
			want: false,
		},
		{
			name:   "Developer list is nil",
			fields: fields{developers: nil},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale.example.com",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TenantAccountsFilter{
				developers: tt.fields.developers,
			}
			if got := f.Developer(tt.args.r); got != tt.want {
				t.Errorf("Developer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantAccountsFilter_Provider(t *testing.T) {
	type fields struct {
		providers []AccountDetail
	}
	type args struct {
		r v1.Route
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Url exists and is approved",
			fields: fields{providers: []AccountDetail{
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "dummy-admin.example.com",
				},
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "3scale-admin.example.com",
				},
			}},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale-admin.example.com",
					},
				},
			},
			want: true,
		},
		{
			name: "Url exists and is not approved",
			fields: fields{providers: []AccountDetail{
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "dummy-admin.example.com",
				},
				{
					Id:           0,
					State:        "not approved",
					AdminBaseURL: "3scale-admin.example.com",
				},
			}},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale-admin.example.com",
					},
				},
			},
			want: false,
		},
		{
			name: "Url does not exist",
			fields: fields{providers: []AccountDetail{
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "dummy-admin.example.com",
				},
				{
					Id:           0,
					State:        "approved",
					AdminBaseURL: "dummy2-admin.example.com",
				},
			}},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale-admin.example.com",
					},
				},
			},
			want: false,
		},
		{
			name:   "providers list is nil",
			fields: fields{providers: nil},
			args: args{
				r: v1.Route{
					Spec: v1.RouteSpec{
						Host: "3scale-admin.example.com",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TenantAccountsFilter{
				providers: tt.fields.providers,
			}
			if got := f.Provider(tt.args.r); got != tt.want {
				t.Errorf("Provider() = %v, want %v", got, tt.want)
			}
		})
	}
}
