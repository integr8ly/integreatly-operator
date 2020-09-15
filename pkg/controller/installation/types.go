package installation

import (
	"errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Stage struct {
	Products map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus
	Name     integreatlyv1alpha1.StageName
}

var (
	allManagedApiStages = &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.CloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {
						Name: integreatlyv1alpha1.ProductCloudResources,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.MonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
			{
				Name: integreatlyv1alpha1.AuthenticationStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO: {
						Name: integreatlyv1alpha1.ProductRHSSO,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.ProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.Product3Scale:    {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser: {Name: integreatlyv1alpha1.ProductRHSSOUser},
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
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallCloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallMonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
		},
	}
	allManagedStages = &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.CloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {
						Name: integreatlyv1alpha1.ProductCloudResources,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.MonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
			{
				Name: integreatlyv1alpha1.AuthenticationStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO: {
						Name: integreatlyv1alpha1.ProductRHSSO,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.ProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductFuse:                {Name: integreatlyv1alpha1.ProductFuse},
					integreatlyv1alpha1.ProductFuseOnOpenshift:     {Name: integreatlyv1alpha1.ProductFuseOnOpenshift},
					integreatlyv1alpha1.ProductCodeReadyWorkspaces: {Name: integreatlyv1alpha1.ProductCodeReadyWorkspaces},
					integreatlyv1alpha1.ProductAMQOnline:           {Name: integreatlyv1alpha1.ProductAMQOnline},
					integreatlyv1alpha1.Product3Scale:              {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser:           {Name: integreatlyv1alpha1.ProductRHSSOUser},
					integreatlyv1alpha1.ProductUps:                 {Name: integreatlyv1alpha1.ProductUps},
					integreatlyv1alpha1.ProductApicurito:           {Name: integreatlyv1alpha1.ProductApicurito},
					integreatlyv1alpha1.ProductDataSync:            {Name: integreatlyv1alpha1.ProductDataSync},
				},
			},
			{
				Name: integreatlyv1alpha1.SolutionExplorerStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductSolutionExplorer: {Name: integreatlyv1alpha1.ProductSolutionExplorer},
				},
			},
		},
		[]Stage{
			{
				Name: integreatlyv1alpha1.UninstallProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO:               {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.ProductFuse:                {Name: integreatlyv1alpha1.ProductFuse},
					integreatlyv1alpha1.ProductFuseOnOpenshift:     {Name: integreatlyv1alpha1.ProductFuseOnOpenshift},
					integreatlyv1alpha1.ProductCodeReadyWorkspaces: {Name: integreatlyv1alpha1.ProductCodeReadyWorkspaces},
					integreatlyv1alpha1.ProductAMQOnline:           {Name: integreatlyv1alpha1.ProductAMQOnline},
					integreatlyv1alpha1.Product3Scale:              {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser:           {Name: integreatlyv1alpha1.ProductRHSSOUser},
					integreatlyv1alpha1.ProductUps:                 {Name: integreatlyv1alpha1.ProductUps},
					integreatlyv1alpha1.ProductApicurito:           {Name: integreatlyv1alpha1.ProductApicurito},
					integreatlyv1alpha1.ProductDataSync:            {Name: integreatlyv1alpha1.ProductDataSync},
					integreatlyv1alpha1.ProductSolutionExplorer:    {Name: integreatlyv1alpha1.ProductSolutionExplorer},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallCloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallMonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
		},
	}
	allWorkshopStages = &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.CloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {
						Name: integreatlyv1alpha1.ProductCloudResources,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.MonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
			{
				Name: integreatlyv1alpha1.AuthenticationStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO: {
						Name: integreatlyv1alpha1.ProductRHSSO,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.ProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductFuse:                {Name: integreatlyv1alpha1.ProductFuse},
					integreatlyv1alpha1.ProductFuseOnOpenshift:     {Name: integreatlyv1alpha1.ProductFuseOnOpenshift},
					integreatlyv1alpha1.ProductCodeReadyWorkspaces: {Name: integreatlyv1alpha1.ProductCodeReadyWorkspaces},
					integreatlyv1alpha1.ProductAMQOnline:           {Name: integreatlyv1alpha1.ProductAMQOnline},
					integreatlyv1alpha1.Product3Scale:              {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser:           {Name: integreatlyv1alpha1.ProductRHSSOUser},
					integreatlyv1alpha1.ProductUps:                 {Name: integreatlyv1alpha1.ProductUps},
					integreatlyv1alpha1.ProductApicurito:           {Name: integreatlyv1alpha1.ProductApicurito},
					integreatlyv1alpha1.ProductDataSync:            {Name: integreatlyv1alpha1.ProductDataSync},
				},
			},
			{
				Name: integreatlyv1alpha1.SolutionExplorerStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductSolutionExplorer: {Name: integreatlyv1alpha1.ProductSolutionExplorer},
				},
			},
		},
		[]Stage{
			{
				Name: integreatlyv1alpha1.UninstallProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO:               {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.ProductFuse:                {Name: integreatlyv1alpha1.ProductFuse},
					integreatlyv1alpha1.ProductFuseOnOpenshift:     {Name: integreatlyv1alpha1.ProductFuseOnOpenshift},
					integreatlyv1alpha1.ProductCodeReadyWorkspaces: {Name: integreatlyv1alpha1.ProductCodeReadyWorkspaces},
					integreatlyv1alpha1.ProductAMQOnline:           {Name: integreatlyv1alpha1.ProductAMQOnline},
					integreatlyv1alpha1.Product3Scale:              {Name: integreatlyv1alpha1.Product3Scale},
					integreatlyv1alpha1.ProductRHSSOUser:           {Name: integreatlyv1alpha1.ProductRHSSOUser},
					integreatlyv1alpha1.ProductUps:                 {Name: integreatlyv1alpha1.ProductUps},
					integreatlyv1alpha1.ProductApicurito:           {Name: integreatlyv1alpha1.ProductApicurito},
					integreatlyv1alpha1.ProductDataSync:            {Name: integreatlyv1alpha1.ProductDataSync},
					integreatlyv1alpha1.ProductSolutionExplorer:    {Name: integreatlyv1alpha1.ProductSolutionExplorer},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallCloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {Name: integreatlyv1alpha1.ProductCloudResources},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallMonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
		},
	}
	allSelfManagedStages = &Type{
		[]Stage{
			{
				Name: integreatlyv1alpha1.BootstrapStage,
			},
			{
				Name: integreatlyv1alpha1.CloudResourcesStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources: {
						Name: integreatlyv1alpha1.ProductCloudResources,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.MonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
			},
			{
				Name: integreatlyv1alpha1.AuthenticationStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductRHSSO: {
						Name: integreatlyv1alpha1.ProductRHSSO,
					},
				},
			},
			{
				Name: integreatlyv1alpha1.ProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductAMQStreams: {Name: integreatlyv1alpha1.ProductAMQStreams},
				},
			},
			{
				Name: integreatlyv1alpha1.SolutionExplorerStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductSolutionExplorer: {Name: integreatlyv1alpha1.ProductSolutionExplorer},
				},
			},
		},
		[]Stage{
			{
				Name: integreatlyv1alpha1.UninstallProductsStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductCloudResources:   {Name: integreatlyv1alpha1.ProductCloudResources},
					integreatlyv1alpha1.ProductRHSSO:            {Name: integreatlyv1alpha1.ProductRHSSO},
					integreatlyv1alpha1.ProductAMQStreams:       {Name: integreatlyv1alpha1.ProductAMQStreams},
					integreatlyv1alpha1.ProductSolutionExplorer: {Name: integreatlyv1alpha1.ProductSolutionExplorer},
				},
			},
			{
				Name: integreatlyv1alpha1.UninstallMonitoringStage,
				Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
					integreatlyv1alpha1.ProductMonitoring:     {Name: integreatlyv1alpha1.ProductMonitoring},
					integreatlyv1alpha1.ProductMonitoringSpec: {Name: integreatlyv1alpha1.ProductMonitoringSpec},
				},
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

//GetInstallStages returns indexed arrays of products names this is worked through starting at 0
//the install will not move to the next index until all installs in the current index have completed successfully
func (t *Type) GetInstallStages() []Stage {
	return t.InstallStages
}

func (t *Type) GetUninstallStages() []Stage {
	return t.UninstallStages
}

func TypeFactory(installationType string) (*Type, error) {
	//TODO: export this logic to a configmap for each installation type
	switch installationType {
	case string(integreatlyv1alpha1.InstallationTypeWorkshop):
		return newWorkshopType(), nil
	case string(integreatlyv1alpha1.InstallationTypeManaged):
		return newManagedType(), nil
	case string(integreatlyv1alpha1.InstallationTypeManagedApi):
		return newManagedApiType(), nil
	case string(integreatlyv1alpha1.InstallationTypeSelfManaged):
		return newSelfManagedType(), nil
	default:
		return nil, errors.New("unknown installation type: " + installationType)
	}
}

func newWorkshopType() *Type {
	return allWorkshopStages
}

func newManagedType() *Type {
	return allManagedStages
}

func newManagedApiType() *Type {
	return allManagedApiStages
}

func newSelfManagedType() *Type {
	return allSelfManagedStages
}
