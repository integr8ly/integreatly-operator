package installation

import (
	"errors"

	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"
)

var (
	allManagedOrder = [][]v1alpha1.ProductName{
		{v1alpha1.ProductRHSSO}, {v1alpha1.ProductFuse, v1alpha1.ProductCodeReadyWorkspaces, v1alpha1.ProductAMQOnline, v1alpha1.Product3Scale},
	}
	allWorkspaceOrder = [][]v1alpha1.ProductName{
		{v1alpha1.ProductRHSSO}, {v1alpha1.ProductFuse, v1alpha1.ProductCodeReadyWorkspaces, v1alpha1.ProductAMQStreams, v1alpha1.ProductAMQOnline, v1alpha1.Product3Scale},
	}
)

type Type struct {
	productOrder [][]v1alpha1.ProductName
}

func (t *Type) HasProduct(product string) bool {
	return false
}

//GetProductOrder returns indexed arrays of products names this is worked through starting at 0
//the install will not move to the next index until all installs in the current index have completed successfully
func (t *Type) GetProductOrder() [][]v1alpha1.ProductName {
	return t.productOrder
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
		productOrder: [][]v1alpha1.ProductName{
			{},
			{},
		},
	}

	buildProducts(t, products, v1alpha1.InstallationTypeWorkshop)
	return t
}

func newManagedType(products []string) *Type {
	logrus.Info("installing managed products ", products)
	t := &Type{
		productOrder: [][]v1alpha1.ProductName{
			{},
			{},
		},
	}
	buildProducts(t, products, v1alpha1.InstallationTypeManaged)
	return t
}

func buildProducts(t *Type, products []string, installType v1alpha1.InstallationType) {
	for _, p := range products {
		product := strings.ToLower(strings.TrimSpace(p))
		if product == "all" {
			if installType == v1alpha1.InstallationTypeManaged {
				t.productOrder = allManagedOrder
			} else if installType == v1alpha1.InstallationTypeWorkshop {
				t.productOrder = allWorkspaceOrder
			}
			break
		}
		if v1alpha1.ProductName(product) == v1alpha1.ProductRHSSO {
			t.productOrder[0] = []v1alpha1.ProductName{v1alpha1.ProductRHSSO}
		}
		t.productOrder[1] = append(t.productOrder[1], v1alpha1.ProductName(product))
	}
}
