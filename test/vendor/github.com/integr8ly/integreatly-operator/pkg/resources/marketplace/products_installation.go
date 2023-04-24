package marketplace

import (
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ProductsInstallation contains the declaration of all product operators and
// how to install them
type ProductsInstallation struct {
	Products ProductsDeclaration `yaml:"products"`
}

type ProductsDeclaration map[string]ProductDeclaration

// ProductDeclaration specifies how to install a product operator, either via
// local manifests, or an index image
type ProductDeclaration struct {
	// Where to install the product from. Either "local", or "index"
	InstallFrom ProductInstallationSource `yaml:"installFrom"`
	// If InstallFrom is "local", the directory where the manifests for this
	// product is stored
	ManifestsDir *string `yaml:"manifestsDir,omitempty"`
	// If InstallFrom is "index", the tag of the index image that serves
	// the manifests
	Index string `yaml:"index,omitempty"`
	// Channel to install the product. Defaults to `rhmi`
	Channel string `yaml:"channel"`
	// Name of the package that provides the product
	Package string `yaml:"package,omitempty"`
}

type ProductInstallationSource string

var ProductInstallationSourceLocal ProductInstallationSource = "local"
var ProductInstallationSourceIndex ProductInstallationSource = "index"
var ProductInstallationSourceImplicit ProductInstallationSource = "implicit"

func LocalProductDeclaration(manifestsPath string) *ProductDeclaration {
	manifestsDir := fmt.Sprintf("manifests/%s", manifestsPath)

	return &ProductDeclaration{
		InstallFrom:  ProductInstallationSourceLocal,
		ManifestsDir: &manifestsDir,
	}
}

// ToCatalogSourceReconciler creates a `CatalogSourceReconciler` instance that
// reconciles the correct CatalogSource using the specification from p
func (p *ProductDeclaration) ToCatalogSourceReconciler(log logger.Logger, client k8sclient.Client, namespace, catalogSourceName string) (CatalogSourceReconciler, error) {
	switch p.InstallFrom {
	case ProductInstallationSourceIndex:
		return NewGRPCImageCatalogSourceReconciler(p.Index, client, namespace, catalogSourceName, log), nil
	case ProductInstallationSourceLocal:
		if p.ManifestsDir == nil {
			return nil, fmt.Errorf("installation source %s requires manifestsDir", p.InstallFrom)
		}
		return NewConfigMapCatalogSourceReconciler(*p.ManifestsDir, client, namespace, catalogSourceName), nil
	case ProductInstallationSourceImplicit:
		return NewImplicitCatalogSourceReconciler(log, client)
	}

	return nil, fmt.Errorf("installation source %s not supported", p.InstallFrom)
}

// GetChannel returns the channel for the product declared in p. Default to
// `rhmi` if the field is empty
func (p *ProductDeclaration) GetChannel() string {
	if p.Channel == "" {
		return IntegreatlyChannel
	}

	return p.Channel
}

// GetPackage returns the package for the product declared in p, and whether
// the package was specified or not
func (p *ProductDeclaration) GetPackage() (string, bool) {
	return p.Package, p.Package != ""
}

// PrepareTarget mutates a minimal target to fullfil the installation of the product
// declared by p, and returns a `CatalogSourceReconciler` instance that reconciles
// the CatalogSource that provides the product
func (p *ProductDeclaration) PrepareTarget(log logger.Logger, client k8sclient.Client, catalogSourceName string, target *Target) (CatalogSourceReconciler, error) {
	catalogSourceReconciler, err := p.ToCatalogSourceReconciler(
		log, client, target.Namespace, catalogSourceName,
	)
	if err != nil {
		return nil, err
	}

	channel := p.GetChannel()
	pkg, ok := p.GetPackage()
	if !ok {
		pkg = target.SubscriptionName
	}

	target.Channel = channel
	target.Package = pkg

	return catalogSourceReconciler, nil
}
