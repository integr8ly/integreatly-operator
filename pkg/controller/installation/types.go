package installation

import (
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Stage struct {
	Products map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus
	Name     integreatlyv1alpha1.StageName
}

var (
	allManagedStages = []Stage{
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
				integreatlyv1alpha1.ProductMonitoring: {Name: integreatlyv1alpha1.ProductMonitoring},
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
	}
	allWorkshopStages = []Stage{
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
				integreatlyv1alpha1.ProductMonitoring: {Name: integreatlyv1alpha1.ProductMonitoring},
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
	}
	allSelfManagedStages = []Stage{
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
				integreatlyv1alpha1.ProductMonitoring: {Name: integreatlyv1alpha1.ProductMonitoring},
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
	}
)

type Type struct {
	Stages []Stage
}

func (t *Type) HasProduct(product string) bool {
	return false
}

//GetProductOrder returns indexed arrays of products names this is worked through starting at 0
//the install will not move to the next index until all installs in the current index have completed successfully
func (t *Type) GetStages() []Stage {
	return t.Stages
}

func TypeFactory(installationType string, products []string) (*Type, error) {
	//TODO: export this logic to a configmap for each installation type
	switch installationType {
	case string(integreatlyv1alpha1.InstallationTypeWorkshop):
		return newWorkshopType(products), nil
	case string(integreatlyv1alpha1.InstallationTypeManaged):
		return newManagedType(products), nil
	case string(integreatlyv1alpha1.InstallationTypeSelfManaged):
		return newSelfManagedType(products), nil
	default:
		return nil, errors.New("unknown installation type: " + installationType)
	}
}

func newWorkshopType(products []string) *Type {
	logrus.Info("Reconciling workshop products ", products)
	t := &Type{
		Stages: []Stage{},
	}

	buildProducts(t, products, integreatlyv1alpha1.InstallationTypeWorkshop)
	return t
}

func newManagedType(products []string) *Type {
	logrus.Info("Reconciling managed products ", products)
	t := &Type{
		Stages: []Stage{},
	}
	buildProducts(t, products, integreatlyv1alpha1.InstallationTypeManaged)
	return t
}

func newSelfManagedType(products []string) *Type {
	logrus.Info("Reconciling self-managed products ", products)
	t := &Type{
		Stages: []Stage{},
	}
	buildProducts(t, products, integreatlyv1alpha1.InstallationTypeSelfManaged)
	return t
}

func buildProducts(t *Type, products []string, installType integreatlyv1alpha1.InstallationType) {
	t.Stages = []Stage{
		Stage{
			Name:     integreatlyv1alpha1.BootstrapStage,
			Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{},
		},
		Stage{
			Name:     integreatlyv1alpha1.AuthenticationStage,
			Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{},
		},
		Stage{
			Name:     integreatlyv1alpha1.ProductsStage,
			Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{},
		},
		Stage{
			Name:     integreatlyv1alpha1.SolutionExplorerStage,
			Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{},
		},
	}
	for _, p := range products {
		product := strings.ToLower(strings.TrimSpace(p))
		if product == "all" {
			if installType == integreatlyv1alpha1.InstallationTypeManaged {
				t.Stages = allManagedStages
			} else if installType == integreatlyv1alpha1.InstallationTypeWorkshop {
				t.Stages = allWorkshopStages
			} else if installType == integreatlyv1alpha1.InstallationTypeSelfManaged {
				t.Stages = allSelfManagedStages
			}
			break
		}
		if integreatlyv1alpha1.ProductName(product) == integreatlyv1alpha1.ProductRHSSO {
			t.Stages[1].Products[integreatlyv1alpha1.ProductRHSSO] = integreatlyv1alpha1.RHMIProductStatus{Name: integreatlyv1alpha1.ProductRHSSO}
		}
		if integreatlyv1alpha1.ProductName(product) == integreatlyv1alpha1.ProductSolutionExplorer {
			t.Stages[3].Products[integreatlyv1alpha1.ProductSolutionExplorer] = integreatlyv1alpha1.RHMIProductStatus{Name: integreatlyv1alpha1.ProductSolutionExplorer}
		}

		t.Stages[2].Products[integreatlyv1alpha1.ProductName(product)] = integreatlyv1alpha1.RHMIProductStatus{Name: integreatlyv1alpha1.ProductName(product)}
	}
}
