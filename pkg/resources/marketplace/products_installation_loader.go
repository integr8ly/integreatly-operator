package marketplace

import (
	"os"

	"gopkg.in/yaml.v2"
)

var (
	DefaultProductsInstallationPath = "products/installation.yaml"
)

// ProductsInstallationLoader knows how to retrieve the ProductsInstallation
// instance
type ProductsInstallationLoader interface {
	GetProductsInstallation() (*ProductsInstallation, error)
}

type FSProductsInstallationLoader struct {
	path string
}

var _ ProductsInstallationLoader = &FSProductsInstallationLoader{}

// NewFSProductInstallationLoader creates a ProductsInstallationLoader instance
// that retrieves the ProductsInstallation from the file system as a YAML file
func NewFSProductInstallationLoader(path string) ProductsInstallationLoader {
	return &FSProductsInstallationLoader{
		path: path,
	}
}

func (l *FSProductsInstallationLoader) GetProductsInstallation() (*ProductsInstallation, error) {
	file, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}

	productsInstallation := &ProductsInstallation{}
	err = yaml.Unmarshal(file, productsInstallation)

	return productsInstallation, err
}

func GetProductsInstallationPath() string {
	if path, ok := os.LookupEnv("PRODUCT_DECLARATION"); ok {
		return path
	}

	return DefaultProductsInstallationPath
}
