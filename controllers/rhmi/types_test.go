package controllers

import (
	"context"
	"reflect"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/utils"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconciler_TypeFactory(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	mcgTestStages := &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.InstallStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
					integreatlyv1alpha1.ProductMCG:            {Name: integreatlyv1alpha1.ProductMCG},
					integreatlyv1alpha1.ProductObservability:  {Name: integreatlyv1alpha1.ProductObservability},
					integreatlyv1alpha1.ProductRHSSO:          {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.Product3Scale:         {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser:      {Name: integreatlyv1alpha1.ProductRHSSOUser},
					integreatlyv1alpha1.ProductMarin3r:        {Name: integreatlyv1alpha1.ProductMarin3r},
					integreatlyv1alpha1.ProductGrafana:        {Name: integreatlyv1alpha1.ProductGrafana},
				},
			},
		},
		[]Stage{
			{
				Name: integreatlyv1alpha1.UninstallProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO:     {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.Product3Scale:    {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser: {Name: integreatlyv1alpha1.ProductRHSSOUser},
					integreatlyv1alpha1.ProductMarin3r:   {Name: integreatlyv1alpha1.ProductMarin3r},
					integreatlyv1alpha1.ProductGrafana:   {Name: integreatlyv1alpha1.ProductGrafana},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallCloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMCG:            {Name: integreatlyv1alpha1.ProductMCG},
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallBootstrap,
			},
		},
	}
	type args struct {
		installationType integreatlyv1alpha1.InstallationType
		client           client.Client
	}
	tests := []struct {
		name string
		args args
		want *Type
		err  error
	}{
		{
			name: "default managed api return type",
			args: args{
				client:           fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(buildTestInfra(configv1.AWSPlatformType)).Build(),
				installationType: integreatlyv1alpha1.InstallationTypeManagedApi,
			},
			want: newManagedApiType(),
			err:  nil,
		},
		{
			name: "gcp mcg managed api return type",
			args: args{
				client:           fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(buildTestInfra(configv1.GCPPlatformType)).Build(),
				installationType: integreatlyv1alpha1.InstallationTypeManagedApi,
			},
			want: mcgTestStages,
			err:  nil,
		},
		{
			name: "default multitenant managed api return type",
			args: args{
				client:           fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(buildTestInfra(configv1.AWSPlatformType)).Build(),
				installationType: integreatlyv1alpha1.InstallationTypeMultitenantManagedApi,
			},
			want: newMultitenantManagedApiType(),
			err:  nil,
		},
		{
			name: "error retrieving platform type",
			args: args{
				client:           fake.NewClientBuilder().WithScheme(scheme).Build(),
				installationType: integreatlyv1alpha1.InstallationTypeManagedApi,
			},
			want: nil,
			err:  errors.New("failed to determine platform type:"),
		},
		{
			name: "error unknown installation type",
			args: args{
				installationType: "unknown-type",
			},
			want: nil,
			err:  errors.New("unknown installation type:"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TypeFactory(context.TODO(), string(tt.args.installationType), tt.args.client)
			if err != nil && tt.err != nil && !strings.Contains(err.Error(), tt.err.Error()) {
				t.Errorf("TypeFactory() error = %v, err %v", err, tt.err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TypeFactory() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func buildTestInfra(platformType configv1.PlatformType) *configv1.Infrastructure {
	return &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: configv1.InfrastructureStatus{
			PlatformStatus: &configv1.PlatformStatus{
				Type: platformType,
			},
		},
	}
}
