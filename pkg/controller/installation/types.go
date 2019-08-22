package installation

import (
	"errors"

	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"
)

type Stage struct {
	Products map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus
	Name     string
}

var (
	allManagedStages = []Stage{
		{
			Name: "authentication",
			Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
				v1alpha1.ProductRHSSO: {
					Name: v1alpha1.ProductRHSSO,
				},
			},
		},
		{
			Name: "products",
			Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
				v1alpha1.ProductLauncher:            {Name: v1alpha1.ProductLauncher},
				v1alpha1.ProductFuse:                {Name: v1alpha1.ProductFuse},
				v1alpha1.ProductCodeReadyWorkspaces: {Name: v1alpha1.ProductCodeReadyWorkspaces},
				v1alpha1.ProductAMQOnline:           {Name: v1alpha1.ProductAMQOnline},
				v1alpha1.Product3Scale:              {Name: v1alpha1.Product3Scale},
				v1alpha1.ProductSolutionExplorer:    {Name: v1alpha1.ProductSolutionExplorer},
			},
		},
	}
	allWorkspaceStages = []Stage{
		{
			Name: "authentication",
			Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
				v1alpha1.ProductRHSSO: {
					Name: v1alpha1.ProductRHSSO,
				},
			},
		},
		{
			Name: "products",
			Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
				v1alpha1.ProductLauncher:            {Name: v1alpha1.ProductLauncher},
				v1alpha1.ProductFuse:                {Name: v1alpha1.ProductFuse},
				v1alpha1.ProductCodeReadyWorkspaces: {Name: v1alpha1.ProductCodeReadyWorkspaces},
				v1alpha1.ProductAMQOnline:           {Name: v1alpha1.ProductAMQOnline},
				v1alpha1.Product3Scale:              {Name: v1alpha1.Product3Scale},
				v1alpha1.ProductSolutionExplorer:    {Name: v1alpha1.ProductSolutionExplorer},
				v1alpha1.ProductNexus:               {Name: v1alpha1.ProductNexus},
				v1alpha1.ProductAMQStreams:          {Name: v1alpha1.ProductAMQStreams},
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

func InstallationTypeFactory(installationType string, products []string) (error, *Type) {
	//TODO: export this logic to a configmap for each installation type
	switch installationType {
	case string(v1alpha1.InstallationTypeWorkshop):
		return nil, newWorkshopType(products)
	case string(v1alpha1.InstallationTypeManaged):
		return nil, newManagedType(products)
	default:
		return errors.New("unknown installation type: " + installationType), nil
	}
}

func newWorkshopType(products []string) *Type {
	logrus.Info("installing workshop products ", products)
	t := &Type{
		Stages: []Stage{},
	}

	buildProducts(t, products, v1alpha1.InstallationTypeWorkshop)
	return t
}

func newManagedType(products []string) *Type {
	logrus.Info("installing managed products ", products)
	t := &Type{
		Stages: []Stage{},
	}
	buildProducts(t, products, v1alpha1.InstallationTypeManaged)
	return t
}

func buildProducts(t *Type, products []string, installType v1alpha1.InstallationType) {
	t.Stages = []Stage{
		Stage{
			Name:     "authentication",
			Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{},
		},
		Stage{
			Name:     "products",
			Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{},
		},
	}
	for _, p := range products {
		product := strings.ToLower(strings.TrimSpace(p))
		if product == "all" {
			if installType == v1alpha1.InstallationTypeManaged {
				t.Stages = allManagedStages
			} else if installType == v1alpha1.InstallationTypeWorkshop {
				t.Stages = allWorkspaceStages
			}
			break
		}
		if v1alpha1.ProductName(product) == v1alpha1.ProductRHSSO {
			t.Stages[0].Products[v1alpha1.ProductRHSSO] = &v1alpha1.InstallationProductStatus{Name: v1alpha1.ProductRHSSO}
		}

		t.Stages[1].Products[v1alpha1.ProductName(product)] = &v1alpha1.InstallationProductStatus{Name: v1alpha1.ProductName(product)}
	}
}
