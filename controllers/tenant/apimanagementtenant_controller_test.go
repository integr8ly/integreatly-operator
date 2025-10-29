package controllers

import (
	"fmt"
	"reflect"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	usersv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	validTenantName = "dev-tenant"
	validNamespace  = "test-user01-dev"
	validUsername   = "test-user01"
)

func TestTenantReconciler_getAPIManagementTenant(t *testing.T) {
	validTenant := &integreatlyv1alpha1.APIManagementTenant{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIManagementTenant",
			APIVersion: "integreatly.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      validTenantName,
			Namespace: validNamespace,
		},
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		crName      string
		crNamespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *integreatlyv1alpha1.APIManagementTenant
		wantErr bool
	}{
		{
			name: "Test fails when searched tenant doesn't exist",
			fields: fields{
				Client: utils.NewTestClient(scheme),
			},
			args: args{
				crName:      "bad-name",
				crNamespace: "bad-namespace",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test passes when tenant exists",
			fields: fields{
				Client: utils.NewTestClient(scheme, validTenant),
			},
			args: args{
				crName:      validTenantName,
				crNamespace: validNamespace,
			},
			want:    validTenant,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TenantReconciler{
				Client: tt.fields.Client,
				Scheme: scheme,
				log:    logger.Logger{},
			}
			got, err := r.getAPIManagementTenant(tt.args.crName, tt.args.crNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAPIManagementTenant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAPIManagementTenant() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenantReconciler_getUserByTenantNamespace(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	validUser := &usersv1.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: fmt.Sprintf("%s/%s", usersv1.GroupName, usersv1.GroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: validUsername,
		},
	}

	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
	}
	type args struct {
		ns string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *usersv1.User
		wantErr bool
	}{
		{
			name: "Test fails when searched user doesn't exist",
			fields: fields{
				Client: utils.NewTestClient(scheme),
			},
			args: args{
				ns: "bad-username",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test passes when user exists",
			fields: fields{
				Client: utils.NewTestClient(scheme, validUser),
			},
			args: args{
				ns: validNamespace,
			},
			want:    validUser,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &TenantReconciler{
				Client: tt.fields.Client,
				Scheme: scheme,
				log:    logger.Logger{},
			}
			got, err := r.getUserByTenantNamespace(tt.args.ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUserByTenantNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserByTenantNamespace() got = %v, want %v", got, tt.want)
			}
		})
	}
}
