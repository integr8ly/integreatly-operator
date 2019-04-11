package installation

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
)

type Type struct {
	products     []v1alpha1.ProductName
	productOrder map[int][]v1alpha1.ProductName
}

func (t *Type) GetProducts() []v1alpha1.ProductName {
	return t.products
}

func (t *Type) HasProduct(product string) bool {
	return false
}

//GetProductOrder returns indexed arrays of products names this is worked through starting at 0
//the install will not move to the next index until all installs in the current index have completed successfully
func (t *Type) GetProductOrder() map[int][]v1alpha1.ProductName {
	return t.productOrder
}

func InstallationTypeFactory(installationType string) (error, *Type) {
	//TODO: export this logic to a configmap for each installation type
	switch installationType {
	case string(v1alpha1.InstallationTypeWorkshop):
		return nil, newWorkshopType()
	case string(v1alpha1.InstallationTypeManaged):
		return nil, newManagedType()
	default:
		return errors.New("unknown installation type: " + installationType), nil
	}
}

func newWorkshopType() *Type {
	return &Type{
		products:     []v1alpha1.ProductName{},
		productOrder: map[int][]v1alpha1.ProductName{
			1: {v1alpha1.ProductAMQStreams},
		},
	}
}

func newManagedType() *Type {
	return &Type{
		products: []v1alpha1.ProductName{v1alpha1.ProductAMQStreams},
		productOrder: map[int][]v1alpha1.ProductName{
		},
	}
}
