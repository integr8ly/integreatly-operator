package common

import (
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"

	goctx "context"

	routev1 "github.com/openshift/api/route/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	threeScaleRoutes = []ExpectedRoute{
		ExpectedRoute{
			Name:  "backend",
			isTLS: true,
		},
		ExpectedRoute{
			Name:            "zync-3scale-api-",
			IsGeneratedName: true,
			isTLS:           true,
			ServiceName:     "apicast-staging",
		},
		ExpectedRoute{
			Name:            "zync-3scale-api-",
			IsGeneratedName: true,
			isTLS:           true,
			ServiceName:     "apicast-production",
		},
		ExpectedRoute{
			Name:            "zync-3scale-master-",
			IsGeneratedName: true,
			isTLS:           true,
			ServiceName:     "system-master",
		},
		ExpectedRoute{
			Name:            "zync-3scale-provider-",
			IsGeneratedName: true,
			isTLS:           true,
			ServiceName:     "system-developer",
		},
		ExpectedRoute{
			Name:            "zync-3scale-provider-",
			IsGeneratedName: true,
			isTLS:           true,
			ServiceName:     "system-provider",
		},
	}

	observabilityRoutes = []ExpectedRoute{
		ExpectedRoute{
			Name:  "alertmanager",
			isTLS: true,
		},
		ExpectedRoute{
			Name:  "grafana-route",
			isTLS: true,
		},
		ExpectedRoute{
			Name:  "prometheus",
			isTLS: true,
		},
	}

	rhssoRoutes = []ExpectedRoute{
		ExpectedRoute{
			Name:  "keycloak",
			isTLS: true,
		},
		ExpectedRoute{
			Name:  "keycloak-edge",
			isTLS: true,
		},
	}

	rhoamUserSsoRoutes = []ExpectedRoute{
		ExpectedRoute{
			Name:  "keycloak",
			isTLS: true,
		},
	}

	customerGrafanaRoutes = []ExpectedRoute{
		{
			Name:  "grafana-route",
			isTLS: true,
		},
	}

	mcgRoutes = []ExpectedRoute{
		ExpectedRoute{
			Name:  "noobaa-mgmt",
			isTLS: true,
		},
		ExpectedRoute{
			Name:  "s3",
			isTLS: true,
		},
		ExpectedRoute{
			Name:  "sts",
			isTLS: true,
		},
	}
)

var managedApiExpectedRoutes = map[string][]ExpectedRoute{
	"3scale":                       threeScaleRoutes,
	"observability":                observabilityRoutes,
	"rhsso":                        rhssoRoutes,
	"user-sso":                     rhoamUserSsoRoutes,
	"customer-monitoring-operator": customerGrafanaRoutes,
}

var mtManagedApiExpectedRoutes = map[string][]ExpectedRoute{
	"3scale":                       threeScaleRoutes,
	"observability":                observabilityRoutes,
	"rhsso":                        rhssoRoutes,
	"customer-monitoring-operator": customerGrafanaRoutes,
}

// TestIntegreatlyRoutesExist tests that the routes for all the products are created
func TestIntegreatlyRoutesExist(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	expectedRoutes := getExpectedRoutes(rhmi.Spec.Type, ctx)

	for product, routes := range expectedRoutes {
		for _, expectedRoute := range routes {
			foundRoute, err := getRoute(t, ctx, product, expectedRoute)
			if err != nil {
				t.Errorf("Failed checking route %v for product %v: %v",
					expectedRoute.Name, product, err)

				continue
			}

			foundTLS := isTLS(foundRoute)
			if foundTLS != expectedRoute.isTLS {
				t.Errorf("Failed checking route %s for product %s: Expected TLS to be %v but got %v",
					expectedRoute.Name, product, expectedRoute.isTLS, foundTLS)
			}
		}
	}
}

func getExpectedRoutes(installType string, ctx *TestingContext) map[string][]ExpectedRoute {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		if GetPlatformType(ctx) == string(configv1.GCPPlatformType) {
			mtManagedApiExpectedRoutes["mcg-operator"] = mcgRoutes
		}
		return mtManagedApiExpectedRoutes
	} else {
		if GetPlatformType(ctx) == string(configv1.GCPPlatformType) {
			managedApiExpectedRoutes["mcg-operator"] = mcgRoutes
		}
		return managedApiExpectedRoutes
	}
}

func getRoute(t TestingTB, ctx *TestingContext, product string, expectedRoute ExpectedRoute) (*routev1.Route, error) {
	if expectedRoute.IsGeneratedName {
		return getRouteByGeneratedName(t, ctx, product, expectedRoute)
	}

	return getRouteByName(t, ctx, product, expectedRoute)
}

// getRouteByName finds a Route by searching for a route that has a matching name
// to expectedRoute.Name
func getRouteByName(t TestingTB, ctx *TestingContext, product string, expectedRoute ExpectedRoute) (*routev1.Route, error) {
	route := &routev1.Route{}
	err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: expectedRoute.Name, Namespace: NamespacePrefix + product}, route)

	if err != nil {
		return nil, err
	}

	return route, nil
}

// getRouteByGeneratedName finds a Route by querying the product routes and finding the
// first route with a generated name that matches expectedRoute.Name and pointing
// to a service that matches expectedRoute.ServiceName
func getRouteByGeneratedName(t TestingTB, ctx *TestingContext, product string, expectedRoute ExpectedRoute) (*routev1.Route, error) {
	routes := &routev1.RouteList{}

	// Get the routes for the product
	err := ctx.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{
		Namespace: NamespacePrefix + product,
	})

	if err != nil {
		return nil, fmt.Errorf("Error obtaining routes for product %s", product)
	}

	// Iterate through the routes and look for the one that matches the expected one.
	// If it's found, check if TLS matches the expected
	for _, route := range routes.Items {
		// Skip routes that don't point to services
		if route.Spec.To.Kind != "Service" {
			continue
		}

		generatedName := route.GetObjectMeta().GetGenerateName()
		to := route.Spec.To.Name

		if expectedRoute.Name == generatedName && expectedRoute.ServiceName == to {
			return &route, nil
		}
	}

	// The loop finished and the expected route wasn't found, return an error
	return nil, fmt.Errorf("Expected route with generated name %v to service %v was not found for product %v",
		expectedRoute.Name, expectedRoute.ServiceName, product)
}

func isTLS(route *routev1.Route) bool {
	return route.Spec.TLS.Termination == routev1.TLSTerminationEdge ||
		route.Spec.TLS.Termination == routev1.TLSTerminationReencrypt
}
