package marketplace

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPrepareTarget(t *testing.T) {
	type assertionFunc func(Target, CatalogSourceReconciler, error) error

	// Utility meta assertions
	all := func(assertions ...assertionFunc) assertionFunc {
		return func(t Target, csr CatalogSourceReconciler, e error) error {
			for _, assertion := range assertions {
				if err := assertion(t, csr, e); err != nil {
					return err
				}
			}

			return nil
		}
	}
	noError := func(_ Target, _ CatalogSourceReconciler, err error) error {
		if err != nil {
			return fmt.Errorf("unexpected error: %v", err)
		}
		return nil
	}
	targetEquals := func(expected Target) assertionFunc {
		return func(t Target, _ CatalogSourceReconciler, _ error) error {
			if !reflect.DeepEqual(t, expected) {
				return errors.New("mutated target doesn't match expected")
			}

			return nil
		}
	}
	createsGRPCReconciler := func(image, namespace, csName string) assertionFunc {
		return func(_ Target, csr CatalogSourceReconciler, _ error) error {
			r, ok := csr.(*GRPCImageCatalogSourceReconciler)
			if !ok {
				return errors.New("unexpected type for CatalogSourceReconciler. Expected GRPCImageCatalogSourceReconciler")
			}

			if r.Image != image {
				return fmt.Errorf("unexpected image. Expected %s, got %s", image, r.Image)
			}
			if r.Namespace != namespace {
				return fmt.Errorf("unexpected namespace. Expected %s, got %s", namespace, r.Namespace)
			}
			if r.CSName != csName {
				return fmt.Errorf("unexpected CatalogSource name. Expected %s, got %s", csName, r.CSName)
			}

			return nil
		}
	}
	createsConfigMapReconciler := func(manifestsDir, namespace, csName string) assertionFunc {
		return func(_ Target, csr CatalogSourceReconciler, _ error) error {
			r, ok := csr.(*ConfigMapCatalogSourceReconciler)
			if !ok {
				return errors.New("unexpected type for CatalogSourceReconciler. Expected ConfigMapCatalogSourceReconciler")
			}

			if r.ManifestsProductDirectory != manifestsDir {
				return fmt.Errorf("unexpected manifests dir. Expected %s, got %s", manifestsDir, r.ManifestsProductDirectory)
			}
			if r.Namespace != namespace {
				return fmt.Errorf("unexpected namespace. Expected %s, got %s", namespace, r.Namespace)
			}
			if r.CSName != csName {
				return fmt.Errorf("unexpected CatalogSource name. Expected %s, got %s", csName, r.CSName)
			}

			return nil
		}
	}

	manifestsDir := func(s string) *string {
		return &s
	}

	scenarios := []struct {
		Name string

		ProductDeclaration ProductDeclaration
		Target             Target
		CatalogSourceName  string

		Assertion assertionFunc
	}{
		{
			Name: "Local declaration",
			ProductDeclaration: ProductDeclaration{
				InstallFrom:  ProductInstallationSourceLocal,
				ManifestsDir: manifestsDir("manifests/test"),
				Channel:      "test-channel",
				Package:      "test-package",
			},
			Target: Target{
				Namespace:        "test-namespace",
				SubscriptionName: "test-product",
			},
			CatalogSourceName: "test-cs",
			Assertion: all(
				noError,
				targetEquals(Target{
					Namespace:        "test-namespace",
					SubscriptionName: "test-product",
					Package:          "test-package",
					Channel:          "test-channel",
				}),
				createsConfigMapReconciler(
					"manifests/test",
					"test-namespace",
					"test-cs",
				),
			),
		},

		{
			Name: "Index declaration",
			ProductDeclaration: ProductDeclaration{
				InstallFrom: ProductInstallationSourceIndex,
				Index:       "quay.io/test/index",
				Channel:     "test-channel",
				Package:     "test-package",
			},
			Target: Target{
				Namespace:        "test-namespace",
				SubscriptionName: "test-product",
			},
			CatalogSourceName: "test-cs",
			Assertion: all(
				noError,
				targetEquals(Target{
					Namespace:        "test-namespace",
					SubscriptionName: "test-product",
					Package:          "test-package",
					Channel:          "test-channel",
				}),
				createsGRPCReconciler(
					"quay.io/test/index",
					"test-namespace",
					"test-cs",
				),
			),
		},

		{
			Name: "Default channel and package",
			ProductDeclaration: ProductDeclaration{
				InstallFrom: ProductInstallationSourceIndex,
				Index:       "quay.io/test/index",
			},
			Target: Target{
				Namespace:        "test-namespace",
				SubscriptionName: "test-product",
			},
			CatalogSourceName: "test-cs",
			Assertion: all(
				noError,
				targetEquals(Target{
					Namespace:        "test-namespace",
					SubscriptionName: "test-product",
					// Package is set from SubscriptionName
					Package: "test-product",
					// Channel defaults to `rhmi`
					Channel: "rhmi",
				}),
			),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Client should not be used. Empty client will suffice
			client := utils.NewTestClient(runtime.NewScheme())

			target := &scenario.Target

			csReconciler, err := scenario.ProductDeclaration.PrepareTarget(
				logger.NewLogger(),
				client,
				scenario.CatalogSourceName,
				target,
			)

			if assertionError := scenario.Assertion(*target, csReconciler, err); assertionError != nil {
				t.Fatal(assertionError)
			}
		})
	}
}
