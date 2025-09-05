package controllers

import (
	"errors"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
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

func TypeFactory(installationType string) (*Type, error) {
	//TODO: export this logic to a configmap for each installation type
	switch installationType {
	case string(integreatlyv1alpha1.InstallationTypeManagedApi):
		return newManagedApiType(), nil
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

// GetProductNamespaces builds the expected product namespaces for all possible RHOAM products
func GetProductNamespaces(watchNamespace string) (map[string]cache.Config, error) {
	namespacePrefix := strings.TrimSuffix(watchNamespace, "operator")
	productNamespaces := map[string]cache.Config{
		watchNamespace: {},
	}

	expectedNamespaces := []string{
		string(integreatlyv1alpha1.Product3Scale),
		string(integreatlyv1alpha1.Product3Scale) + "-operator",
		string(integreatlyv1alpha1.ProductMarin3r),
		string(integreatlyv1alpha1.ProductMarin3r) + "-operator",
		string(integreatlyv1alpha1.ProductCloudResources) + "-operator",
		string(integreatlyv1alpha1.ProductRHSSO),
		string(integreatlyv1alpha1.ProductRHSSO) + "-operator",
		// namespaces different from the product name
		"user-sso",
		"user-sso-operator",
		"customer-monitoring",
		"operator-observability",
	}

	// Add all expected namespaces with the prefix
	for _, suffix := range expectedNamespaces {
		namespace := namespacePrefix + suffix
		productNamespaces[namespace] = cache.Config{}
	}

	return productNamespaces, nil
}
