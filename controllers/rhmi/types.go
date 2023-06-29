package controllers

import (
	"context"
	"errors"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/cluster"
	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Stage struct {
	Products map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus
	Name     integreatlyv1alpha1.StageName
}

var (
	allMultitenantManagedApiStages = &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.InstallStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
					integreatlyv1alpha1.ProductObservability:  {Name: integreatlyv1alpha1.ProductObservability}, // TODO MGDAPI-5833
					integreatlyv1alpha1.ProductRHSSO:          {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.Product3Scale:         {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductMarin3r:        {Name: integreatlyv1alpha1.ProductMarin3r},
					integreatlyv1alpha1.ProductGrafana:        {Name: integreatlyv1alpha1.ProductGrafana},
				},
			},
		},
		[]Stage{
			{
				Name: integreatlyv1alpha1.UninstallProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO:   {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.Product3Scale:  {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductMarin3r: {Name: integreatlyv1alpha1.ProductMarin3r},
					integreatlyv1alpha1.ProductGrafana: {Name: integreatlyv1alpha1.ProductGrafana},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallCloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallBootstrap,
			},
		},
	}
	allManagedApiStages = &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.InstallStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
					integreatlyv1alpha1.ProductObservability:  {Name: integreatlyv1alpha1.ProductObservability}, // TODO MGDAPI-5833
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
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallBootstrap,
			},
		},
	}
)

type Type struct {
	InstallStages   []Stage
	UninstallStages []Stage
}

func (t *Type) HasProduct(product string) bool {
	return false
}

// GetInstallStages returns indexed arrays of products names this is worked through starting at 0
// the install will not move to the next index until all installs in the current index have completed successfully
func (t *Type) GetInstallStages() []Stage {
	return t.InstallStages
}

func (t *Type) GetUninstallStages() []Stage {
	return t.UninstallStages
}

func TypeFactory(ctx context.Context, installationType string, c client.Client) (*Type, error) {
	//TODO: export this logic to a configmap for each installation type
	switch installationType {
	case string(integreatlyv1alpha1.InstallationTypeManagedApi):
		platform, err := cluster.GetPlatformType(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to determine platform type: %v", err)
		}
		managedApiType := newManagedApiType()
		if platform == configv1.GCPPlatformType {
			for i := range managedApiType.InstallStages {
				if managedApiType.InstallStages[i].Name == integreatlyv1alpha1.InstallStage {
					managedApiType.InstallStages[i].Products[integreatlyv1alpha1.ProductMCG] = integreatlyv1alpha1.RHMIProductStatus{Name: integreatlyv1alpha1.ProductMCG}
					break
				}
			}
			for i := range managedApiType.UninstallStages {
				if managedApiType.UninstallStages[i].Name == integreatlyv1alpha1.UninstallCloudResourcesStage {
					managedApiType.UninstallStages[i].Products[integreatlyv1alpha1.ProductMCG] = integreatlyv1alpha1.RHMIProductStatus{Name: integreatlyv1alpha1.ProductMCG}
					break
				}
			}
		}
		return managedApiType, nil
	case string(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi):
		return newMultitenantManagedApiType(), nil
	default:
		return nil, errors.New("unknown installation type: " + installationType)
	}
}

func newManagedApiType() *Type {
	return allManagedApiStages
}

func newMultitenantManagedApiType() *Type {
	return allMultitenantManagedApiStages
}
