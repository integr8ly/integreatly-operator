package common

import (
	"context"
	goctx "context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	portaclient "github.com/3scale/3scale-porta-go-client/client"
	"github.com/headzoo/surf"
	brow "github.com/headzoo/surf/browser"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	pollingTime          = time.Second * 5
	tenantReadyTimeout   = time.Minute * 10
	userReadyTimeout     = time.Minute * 5
	resourceReadyTimeout = time.Minute * 2
	expectedEndpointResp = `[{"name":"Apple","description":"Winter fruit"},{"name":"Pineapple","description":"Tropical fruit"}]`
	testUserPoolSize     = 100
)

var (
	testUserForDeletion   = fmt.Sprintf("%v%02v", DefaultTestUserName, 1)
	testUserForQuickStart = fmt.Sprintf("%v%02v", DefaultTestUserName, 1)
	quickStartNamespace   = fmt.Sprintf("%v%02v-dev", DefaultTestUserName, 1)
	quarkusAppName        = "rhoam-quarkus-openapi"
	quarkusImageName      = "quay.io/integreatly/rhoam-quarkus-openapi:latest"
	singleTenantMode      = true
)

func TestMultitenancy(t TestingTB, ctx *TestingContext) {
	// Get master URL
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Testing IDP with 2 regular users and 2 admins gets created
	err = createTestingIDP(t, context.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts)
	if err != nil {
		t.Errorf("error while creating testing IDP: %s", err)
	}

	// Check to see if test should be run in singleTenantMode or if multiple tenants should be created
	numTenants := getNumberOfTestUsersFromEnv()
	if numTenants == 1 {
		// Make sure the test-user doesn't already exist in 3scale and if it does, create a new test-user
		// This is required because 3scale waits a few weeks before it actually removes accounts marked for deletion
		tsName, tsNamespace, err := validateTestUser(ctx, rhmi, testUserForQuickStart)
		if err != nil {
			t.Error(err)
		}
		testUserForDeletion = tsName
		testUserForQuickStart = tsName
		quickStartNamespace = tsNamespace
	} else {
		singleTenantMode = false
	}

	// Verify that the regular users can log in to their 3scale account
	err = loginUsersTo3scale(t, ctx, rhmi, singleTenantMode)
	if err != nil {
		t.Errorf("User login to 3scale failed: %v", err)
	}

	// Import Quarkus container image
	err = importQuarkusImage(ctx)
	if err != nil {
		t.Error(err)
	}

	// Create a Quarkus deployment using the imported image
	err = createQuarkusDeployment(ctx)
	if err != nil {
		t.Error(err)
	}

	// Create service for Quarkus
	err = createQuarkusService(ctx)
	if err != nil {
		t.Error(err)
	}

	// Create route for Quarkus service
	err = createAndVerifyQuarkusRoute(ctx, rhmi)
	if err != nil {
		t.Error(err)
	}

	// Grant permissions to test user to allow for Service Discovery
	err = grantViewRoleToTestUser(ctx)
	if err != nil {
		t.Error(err)
	}

	// Create a 3scale portaClient for test-user
	tsClient, err := createPortaClient(ctx, rhmi, testUserForQuickStart)
	if err != nil {
		t.Error(err)
	}

	// Create 3scale product
	productID, err := createThreescaleProduct(tsClient)
	if err != nil {
		t.Error(err)
	}

	// Create 3scale ActiveDocs
	err = createThreescaleActiveDocs(tsClient, productID)
	if err != nil {
		t.Error(err)
	}

	// Create 3scale backend
	backendID, err := createThreescaleBackend(tsClient)
	if err != nil {
		t.Error(err)
	}

	// Create a 3scale backend usage to bind the product and backend
	err = createThreescaleBackendUsage(tsClient, productID, backendID)
	if err != nil {
		t.Error(err)
	}

	// Promote the proxy configuration to the staging environment now that backend is linked to product
	_, err = tsClient.DeployProductProxy(productID)
	if err != nil {
		t.Error(err)
	}

	// Create a 3scale application plan
	applicationPlanID, err := createThreescaleApplicationPlan(tsClient, productID)
	if err != nil {
		t.Error(err)
	}

	// Create a 3scale application
	userKey, err := createThreescaleApplication(tsClient, applicationPlanID)
	if err != nil {
		t.Error(err)
	}

	// Verifies the API is working properly but curling endpoint and comparing response to expected value
	err = verifyApiIsWorking(t, ctx, tsClient, productID, userKey)
	if err != nil {
		t.Error(err)
	}

	// Delete user CR for one of the users
	err = deleteTenantUserCR(t, ctx)
	if err != nil {
		t.Error(err)
	}

	// Confirm that 3scale is no longer available for the deleted tenant
	err = confirmTenantAccountNotAvailable(t, ctx, rhmi.Spec.RoutingSubdomain)
	if err != nil {
		t.Error(err)
	}
}

func loginUsersTo3scale(t TestingTB, ctx *TestingContext, rhmi *integreatlyv1alpha1.RHMI, singleTenantMode bool) error {
	postfix := 1
	numberOfTestUsers := getNumberOfTestUsersFromEnv()
	for postfix <= numberOfTestUsers {
		isClusterLoggedIn := false
		isTenantCRCreated := false
		routeFound := false
		isThreeScaleLoggedIn := false

		// Create client for current tenant
		tenantClient, err := NewTestingHTTPClient(ctx.KubeConfig)
		if err != nil {
			return fmt.Errorf("error while creating client for tenant: %v", err)
		}

		testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)
		if singleTenantMode {
			testUser = testUserForQuickStart
		}

		host := fmt.Sprintf("https://%v-admin.%v", testUser, rhmi.Spec.RoutingSubdomain)

		// Create new namespace for APIManagementTenant CR
		err, testUserNamespace := createTestingUserNamespace(t, testUser, ctx)
		if err != nil {
			return fmt.Errorf("error while creating namespace for testing user: %v", err)
		}

		err = wait.PollUntilContextTimeout(context.TODO(), pollingTime, tenantReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
			// login to cluster
			if !isClusterLoggedIn {
				err = loginToCluster(t, tenantClient, rhmi.Spec.MasterURL, testUser)
				if err != nil {
					return false, nil
				} else {
					isClusterLoggedIn = true
				}
			}

			// Create testing user CR an APIManagementTenant CR
			if !isTenantCRCreated {
				err = createTestingUserApiManagementTenantCR(t, testUser, testUserNamespace, ctx)
				if err != nil {
					return false, nil
				} else {
					isTenantCRCreated = true
				}
			}

			// check if 3scale route is available
			if !routeFound {
				err = getTenant3scaleRoute(t, ctx, testUser)
				if err != nil {
					return false, nil
				} else {
					routeFound = true
				}
			}

			// login tenant to 3scale
			if !isThreeScaleLoggedIn {
				err := loginToThreeScale(t, host, testUser, TestingIdpPassword, TestingIDPRealm, tenantClient)
				if err != nil {
					t.Log(fmt.Sprintf("User failed to login to 3scale %s", testUser))
					return false, nil
				} else {
					isThreeScaleLoggedIn = true
				}
			}

			return true, nil
		})

		if err != nil {
			return fmt.Errorf("user %s login and creation of tenant failed with: %v", testUser, err)
		}

		postfix++
	}

	return nil
}

func importQuarkusImage(ctx *TestingContext) error {
	imagestream := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quarkusAppName,
			Namespace: quickStartNamespace,
			Labels: map[string]string{
				"app":                         quarkusAppName,
				"app.kubernetes.io/component": quarkusAppName,
				"app.kubernetes.io/instance":  quarkusAppName,
				"app.kubernetes.io/name":      quarkusAppName,
				"app.kubernetes.io/part-of":   quarkusAppName + "-app",
			},
		},
		Spec: imagev1.ImageStreamSpec{
			Tags: []imagev1.TagReference{
				{
					Name: "latest",
					Annotations: map[string]string{
						"openshift.io/imported-from": quarkusImageName,
					},
					From: &corev1.ObjectReference{
						Kind: "DockerImage",
						Name: quarkusImageName,
					},
					ReferencePolicy: imagev1.TagReferencePolicy{Type: imagev1.LocalTagReferencePolicy},
				},
			},
		},
	}
	_, err := controllerruntime.CreateOrUpdate(goctx.TODO(), ctx.Client, imagestream, func() error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("error while importing quarkus image: %v", err)
	}
	return nil
}

func createQuarkusDeployment(ctx *TestingContext) error {
	// Make sure that ImageStream exists before creating deployment
	err := wait.PollUntilContextTimeout(context.TODO(), pollingTime, resourceReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		// Get the ImageStream successfully to exit the poll
		imageStream := &imagev1.ImageStream{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quarkusAppName,
				Namespace: quickStartNamespace,
			},
		}
		key := k8sclient.ObjectKeyFromObject(imageStream)
		err = ctx.Client.Get(context.TODO(), key, imageStream)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to get ImageStream before creating deployment %v", err)
	}

	// Create the deployment now that the ImageStream is ready
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quarkusAppName,
			Namespace: quickStartNamespace,
			Labels: map[string]string{
				"app":                                quarkusAppName,
				"app.kubernetes.io/component":        quarkusAppName,
				"app.kubernetes.io/instance":         quarkusAppName,
				"app.kubernetes.io/name":             quarkusAppName,
				"app.kubernetes.io/part-of":          quarkusAppName + "-app",
				"app.openshift.io/runtime":           "quarkus",
				"app.openshift.io/runtime-namespace": quickStartNamespace,
			},
			Annotations: map[string]string{
				"alpha.image.policy.openshift.io/resolve-names": "*",
				"image.openshift.io/triggers":                   fmt.Sprintf(`[{"from":{"kind":"ImageStreamTag","name":"rhoam-quarkus-openapi:latest","namespace":"%v"},"fieldPath":"spec.template.spec.containers[?(@.name==\"rhoam-quarkus-openapi\")].image","pause":"false"}]`, quickStartNamespace),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": quarkusAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":              quarkusAppName,
						"deploymentconfig": quarkusAppName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  quarkusAppName,
							Image: fmt.Sprintf("image-registry.openshift-image-registry.svc:5000/%v/%v", quickStartNamespace, quarkusAppName),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ctx.KubeClient.AppsV1().Deployments(quickStartNamespace).Create(goctx.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create quarkus deployment: %v", err)
	}
	return nil
}

func createQuarkusService(ctx *TestingContext) error {
	// Make sure that Deployment exists before creating service
	err := wait.PollUntilContextTimeout(context.TODO(), pollingTime, resourceReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		// Get the Deployment successfully to exit the poll
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quarkusAppName,
				Namespace: quickStartNamespace,
			},
		}
		key := k8sclient.ObjectKeyFromObject(deployment)

		err = ctx.Client.Get(context.TODO(), key, deployment)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to get Deployment before creating service %v", err)
	}

	// Create the service now that the Deployment is ready
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quarkusAppName,
			Namespace: quickStartNamespace,
			Labels: map[string]string{
				"app":                              quarkusAppName,
				"app.kubernetes.io/component":      quarkusAppName,
				"app.kubernetes.io/instance":       quarkusAppName,
				"app.kubernetes.io/name":           quarkusAppName,
				"app.kubernetes.io/part-of":        quarkusAppName + "-app",
				"app.openshift.io/runtime-version": "latest",
				"discovery.3scale.net":             "true",
			},
			Annotations: map[string]string{
				"discovery.3scale.net/description-path": "/q/openapi?format=json",
				"discovery.3scale.net/port":             "8080",
				"discovery.3scale.net/scheme":           "http",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "8080-tcp",
				Port:       int32(8080),
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8080),
			}},
			Selector: map[string]string{
				"app":              quarkusAppName,
				"deploymentconfig": quarkusAppName,
			},
		},
	}
	_, err = ctx.KubeClient.CoreV1().Services(quickStartNamespace).Create(goctx.TODO(), &service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create service for Quarkus: %v", err)
	}
	return nil
}

func createAndVerifyQuarkusRoute(ctx *TestingContext, rhmi *integreatlyv1alpha1.RHMI) error {
	// Make sure that Service exists before creating route
	err := wait.PollUntilContextTimeout(context.TODO(), pollingTime, resourceReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		// Get the Service successfully to exit the poll
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quarkusAppName,
				Namespace: quickStartNamespace,
			},
		}
		key := k8sclient.ObjectKeyFromObject(service)

		err = ctx.Client.Get(context.TODO(), key, service)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to get Service before creating route %v", err)
	}

	// Create the route now that the Service is ready
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      quarkusAppName,
			Namespace: quickStartNamespace,
			Labels: map[string]string{
				"app":                              quarkusAppName,
				"app.kubernetes.io/component":      quarkusAppName,
				"app.kubernetes.io/instance":       quarkusAppName,
				"app.kubernetes.io/name":           quarkusAppName,
				"app.kubernetes.io/part-of":        quarkusAppName + "-app",
				"app.openshift.io/runtime-version": "latest",
			},
			Annotations: map[string]string{
				"openshift.io/host.generated": "true",
			},
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("8080-tcp"),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: quarkusAppName,
			},
			Host: fmt.Sprintf("%v-%v.%v", quarkusAppName, quickStartNamespace, rhmi.Spec.RoutingSubdomain),
		},
	}
	err = ctx.Client.Create(goctx.TODO(), route)
	if err != nil {
		return fmt.Errorf("unable to create route for Quarkus service: %v", err)
	}

	// Verify that the new route was created successfully
	err = wait.PollUntilContextTimeout(context.TODO(), pollingTime, resourceReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		route := &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quarkusAppName,
				Namespace: quickStartNamespace,
			},
		}
		key := k8sclient.ObjectKeyFromObject(route)

		err = ctx.Client.Get(context.TODO(), key, route)
		if err != nil {
			return false, nil
		}

		quarkusPod, err := getQuarkusPod(ctx)
		if err != nil || quarkusPod == nil {
			return false, nil
		}

		output, err := execToPod(fmt.Sprintf("curl --silent %v/q/openapi?format=json", route.Spec.Host),
			quarkusPod.Name,
			quickStartNamespace,
			quarkusAppName,
			ctx)
		if err != nil || output == "" {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to verify that route was successfully created %v", err)
	}

	return nil
}

func grantViewRoleToTestUser(ctx *TestingContext) error {
	// Grant the test user the permissions required by 3scale for Service Discovery
	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user01-view-binding",
			Namespace: quickStartNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, roleBinding, func() error {
		roleBinding.Subjects = []rbac.Subject{
			{
				Kind: rbac.UserKind,
				Name: testUserForQuickStart,
			},
		}
		roleBinding.RoleRef = rbac.RoleRef{
			Kind: "ClusterRole",
			Name: "view",
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to grant view role to user %v, error: %v", testUserForQuickStart, err)
	}

	return nil
}

func createThreescaleProduct(tsClient *portaclient.ThreeScaleClient) (int64, error) {
	params := portaclient.Params{
		"system_name": fmt.Sprintf("%v-%v", quickStartNamespace, quarkusAppName),
	}
	product, err := tsClient.CreateProduct(quarkusAppName, params)
	if err != nil {
		return 0, fmt.Errorf("failed to create 3scale product, error: %v", err)
	}
	if product == nil {
		return 0, fmt.Errorf("created product returned nil")
	}

	return product.Element.ID, nil
}

func createThreescaleActiveDocs(tsClient *portaclient.ThreeScaleClient, serviceID int64) error {
	var (
		published              = true
		skipSwaggerValidations = true
		body                   = `{
  "openapi" : "3.0.3",
  "info" : {
    "title" : "Generated API",
    "version" : "1.0"
  },
  "paths" : {
    "/fruits" : {
      "get" : {
        "responses" : {
          "200" : {
            "description" : "OK",
            "content" : {
              "application/json" : {
                "schema" : {
                  "uniqueItems" : true,
                  "type" : "array",
                  "items" : {
                    "$ref" : "#/components/schemas/Fruit"
                  }
                }
              }
            }
          }
        }
      },
      "post" : {
        "requestBody" : {
          "content" : {
            "application/json" : {
              "schema" : {
                "$ref" : "#/components/schemas/Fruit"
              }
            }
          }
        },
        "responses" : {
          "200" : {
            "description" : "OK",
            "content" : {
              "application/json" : {
                "schema" : {
                  "uniqueItems" : true,
                  "type" : "array",
                  "items" : {
                    "$ref" : "#/components/schemas/Fruit"
                  }
                }
              }
            }
          }
        }
      },
      "delete" : {
        "requestBody" : {
          "content" : {
            "application/json" : {
              "schema" : {
                "$ref" : "#/components/schemas/Fruit"
              }
            }
          }
        },
        "responses" : {
          "200" : {
            "description" : "OK",
            "content" : {
              "application/json" : {
                "schema" : {
                  "uniqueItems" : true,
                  "type" : "array",
                  "items" : {
                    "$ref" : "#/components/schemas/Fruit"
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "components" : {
    "schemas" : {
      "Fruit" : {
        "type" : "object",
        "properties" : {
          "description" : {
            "type" : "string"
          },
          "name" : {
            "type" : "string"
          }
        }
      }
    }
  }
}`
		activeDocToCreate = portaclient.ActiveDoc{
			Element: portaclient.ActiveDocItem{
				ServiceID:              &serviceID,
				Name:                   &quarkusAppName,
				Published:              &published,
				SkipSwaggerValidations: &skipSwaggerValidations,
				Body:                   &body,
			},
		}
	)

	activeDoc, err := tsClient.CreateActiveDoc(&activeDocToCreate)
	if err != nil {
		return fmt.Errorf("failed to create 3scale ActiveDocs, error: %v", err)
	}
	if activeDoc == nil {
		return fmt.Errorf("created ActiveDocs returned nil")
	}

	return nil
}

func createThreescaleBackend(tsClient *portaclient.ThreeScaleClient) (int64, error) {
	params := portaclient.Params{
		"system_name":      fmt.Sprintf("%v-%v", quickStartNamespace, quarkusAppName),
		"name":             fmt.Sprintf("%v Backend", quarkusAppName),
		"description":      fmt.Sprintf("Backend of %v", quarkusAppName),
		"private_endpoint": fmt.Sprintf("http://%v.%v.svc.cluster.local:8080", quarkusAppName, quickStartNamespace),
	}
	backend, err := tsClient.CreateBackendApi(params)
	if err != nil {
		return 0, fmt.Errorf("failed to create 3scale backend, error: %v", err)
	}
	if backend == nil {
		return 0, fmt.Errorf("created backend returned nil")
	}

	return backend.Element.ID, nil
}

func createThreescaleBackendUsage(tsClient *portaclient.ThreeScaleClient, productID int64, backendID int64) error {
	params := portaclient.Params{
		"path":           "/",
		"backend_api_id": strconv.Itoa(int(backendID)),
	}

	backendUsage, err := tsClient.CreateBackendapiUsage(productID, params)
	if err != nil {
		return fmt.Errorf("failed to create 3scale backend usage, error: %v", err)
	}
	if backendUsage == nil {
		return fmt.Errorf("created backend usage returned nil")
	}

	return nil
}

func createThreescaleApplicationPlan(tsClient *portaclient.ThreeScaleClient, productID int64) (int64, error) {
	params := portaclient.Params{
		"name":              "RHOAM Open API Plan",
		"system_name":       "rhoam-openapi-plan",
		"approval_required": "false",
	}
	applicationPlan, err := tsClient.CreateApplicationPlan(productID, params)
	if err != nil {
		return 0, fmt.Errorf("failed to create 3scale application plan, error: %v", err)
	}
	if applicationPlan == nil {
		return 0, fmt.Errorf("created application plan returned nil")
	}

	// The 3scale API doesn't allow for publishing an application plan during it's creation
	// However now that it has been created, we can publish it
	applicationPlan, err = tsClient.UpdateApplicationPlan(productID, applicationPlan.Element.ID, portaclient.Params{"state": "publish"})
	if err != nil {
		return 0, fmt.Errorf("failed to publish 3scale application plan, error: %v", err)
	}
	if applicationPlan == nil {
		return 0, fmt.Errorf("updated application plan returned nil")
	}

	return applicationPlan.Element.ID, nil
}

func createThreescaleApplication(tsClient *portaclient.ThreeScaleClient, planID int64) (string, error) {
	accounts, err := tsClient.ListDeveloperAccounts()
	if err != nil {
		return "", fmt.Errorf("failed to list accounts during 3scale app creation, error: %v", err)
	}
	if accounts == nil {
		return "", fmt.Errorf("failed to get the developer account during 3scale app creation, error: %v", err)
	}
	accountID := accounts.Items[0].Element.ID
	appNameDescription := "Developer RHOAM Application"

	application, err := tsClient.CreateApp(strconv.Itoa(int(*accountID)), strconv.Itoa(int(planID)), appNameDescription, appNameDescription)
	if err != nil {
		return "", fmt.Errorf("failed to create 3scale application, error: %v", err)
	}
	if application.UserKey == "" {
		return "", fmt.Errorf("failed to extract userKey from the 3scale application")
	}

	return application.UserKey, nil
}

func verifyApiIsWorking(t TestingTB, ctx *TestingContext, tsClient *portaclient.ThreeScaleClient, productID int64, userKey string) error {
	// Get the endpoint
	proxy, err := tsClient.ReadProxy(strconv.Itoa(int(productID)))
	if err != nil {
		return fmt.Errorf("failed to get proxy during endpoint verification, error: %v", err)
	}
	if proxy.SandboxEndpoint == "" {
		return fmt.Errorf("failed to parse endpoint from proxy")
	}
	fruitsEndpoint := fmt.Sprintf("%v/fruits/?user_key=%v", proxy.SandboxEndpoint, userKey)

	// Create new http client and query endpoint
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create http client during endpoint verification, error: %v", err)
	}

	req, err := http.NewRequest("GET", fruitsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request during endpoint verification, error: %v", err)
	}

	err = wait.PollUntilContextTimeout(context.TODO(), pollingTime, resourceReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		// Send request, if response code isn't 200 then retry until timeout is reached
		resp, err := httpClient.Do(req)
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Log(fmt.Sprintf("got unexpected status code when querying endpoint; expected: %v, got: %v. Retrying...", http.StatusOK, resp.StatusCode))
			return false, nil
		}

		// Verify that the result matches expected value, fail if it doesn't
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return true, fmt.Errorf("failed to parse http response during endpoint verification, error: %v", err)
		}
		if string(body) != expectedEndpointResp {
			return true, fmt.Errorf("failed to verify endpoint; expected result: %v, got: %v", expectedEndpointResp, body)
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("endpoint verification timedout, error: %v", err)
	}

	return nil
}

func getTenant3scaleRoute(t TestingTB, ctx *TestingContext, testUser string) error {
	routeFound := false
	routes := &routev1.RouteList{}

	err := ctx.Client.List(context.TODO(), routes, &k8sclient.ListOptions{
		Namespace: ThreeScaleProductNamespace,
	})

	if err != nil {
		return err
	}

	for _, route := range routes.Items {
		if route.Spec.To.Kind != "Service" {
			continue
		}
		tenantHostRoute := route.Spec.Host
		if strings.Contains(tenantHostRoute, testUser) {
			routeFound = true
		}
	}

	if !routeFound {
		return fmt.Errorf("route for %s has not yet been found", testUser)
	} else {
		return nil
	}
}

func deleteTenantUserCR(t TestingTB, ctx *TestingContext) error {
	usersList := &usersv1.UserList{}
	err := ctx.Client.List(context.TODO(), usersList)
	if err != nil {
		return fmt.Errorf("failed at finding users list, error: %v", err)
	}

	for i := range usersList.Items {
		user := usersList.Items[i]
		if user.Name == testUserForDeletion {
			err := ctx.Client.Delete(context.TODO(), &user)
			if err != nil {
				return fmt.Errorf("failed to remove user CR, error: %v", err)
			}
		}
	}

	return nil
}

func confirmTenantAccountNotAvailable(t TestingTB, ctx *TestingContext, routingDomain string) error {
	tenantClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		return fmt.Errorf("error while creating client for tenant: %v", err)
	}

	host := fmt.Sprintf("https://%v-admin.%v", testUserForDeletion, routingDomain)

	err = wait.PollUntilContextTimeout(context.TODO(), pollingTime, tenantReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		err = is3scaleLoginFailed(t, host, tenantClient)
		if err != nil {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("3scale did not return 404 after tenant deletion: %v", err)
	}

	return nil
}

func validateTestUser(ctx *TestingContext, rhmi *integreatlyv1alpha1.RHMI, username string) (string, string, error) {
	// Create portaClient to check if any existing 3scale accounts are already using the username
	tsClient, err := createPortaClient(ctx, rhmi, "master")
	if err != nil {
		return "", "", err
	}
	accounts, err := tsClient.ListDeveloperAccounts()
	if err != nil {
		return "", "", fmt.Errorf("failed to list accounts during test user validation, error: %v", err)
	}
	if accounts == nil {
		return "", "", fmt.Errorf("failed to get the developer account during test user validation, error: %v", err)
	}

	usernameToCheck := username
	rnd, err := rand.Int(rand.Reader, big.NewInt(testUserPoolSize))
	if err != nil {
		return "", "", fmt.Errorf("error generating random username")
	}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*1, userReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		for _, account := range accounts.Items {
			if *account.Element.OrgName == usernameToCheck {
				usernameToCheck = fmt.Sprintf("%v%02v", DefaultTestUserName, rnd.Int64())
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to validate test user, error: %v", err)
	}

	// CreateOrUpdate KeycloakUser for validated username
	userNumber, err := strconv.Atoi(strings.TrimPrefix(usernameToCheck, "test-user"))
	if err != nil {
		return "", "", err
	}

	testUsers := []TestUser{
		{
			UserName:  usernameToCheck,
			FirstName: "Test",
			LastName:  fmt.Sprintf("User %v", userNumber),
		},
	}
	err = createOrUpdateKeycloakUserCR(goctx.TODO(), ctx.Client, testUsers, rhmi.Name)
	if err != nil {
		return "", "", fmt.Errorf("error occurred while creating keycloak user for test-user %v, error: %w", usernameToCheck, err)
	}
	return usernameToCheck, fmt.Sprintf("%v-dev", usernameToCheck), nil
}

func loginToCluster(t TestingTB, tenantClient *http.Client, masterURL, testUser string) error {
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), testUser, TestingIdpPassword, tenantClient, TestingIDPRealm, t); err != nil {
		return err
	}
	return nil
}

func is3scaleLoginFailed(t TestingTB, tsHost string, client *http.Client) error {
	parsedURL, err := url.Parse(tsHost)
	if err != nil {
		return fmt.Errorf("failed to parse three scale url %s: %s", parsedURL, err)
	}

	if parsedURL.Scheme == "" {
		tsHost = fmt.Sprintf("https://%s", tsHost)
	}

	tsLoginURL := fmt.Sprintf("%v/p/login", tsHost)
	browser := surf.NewBrowser()
	browser.SetCookieJar(client.Jar)
	browser.SetTransport(client.Transport)
	browser.SetAttribute(brow.FollowRedirects, true)

	err = browser.Open(tsLoginURL)
	if err != nil {
		return err
	}

	statusCode := browser.StatusCode()

	if statusCode != 404 {
		return fmt.Errorf("unsuccessful 3scale login failed, response code recived is %v", statusCode)
	}

	return nil
}

func createTestingUserNamespace(t TestingTB, user string, ctx *TestingContext) (error, string) {
	testUserNamespaceName := user + "-dev"
	testUserNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testUserNamespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, testUserNamespace, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("error while creating namespace for testing user %v", err)
	}

	return nil, testUserNamespaceName
}

func createTestingUserApiManagementTenantCR(t TestingTB, testUserName string, testUserNamespace string, ctx *TestingContext) error {
	// Wait until User has finished being created before attempting to create an APIManagementTenant CR
	err := wait.PollUntilContextTimeout(context.TODO(), pollingTime, userReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		// Get User successfully to exit the poll
		user := &usersv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: testUserName,
			},
		}
		key := k8sclient.ObjectKeyFromObject(user)

		err = ctx.Client.Get(context.TODO(), key, user)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	tenantCR := &integreatlyv1alpha1.APIManagementTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testUserName,
			Namespace: testUserNamespace,
		},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, tenantCR, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("error while creating APIManagementTenant CR for testing user %v", err)
	}
	return nil
}

func getNumberOfTestUsersFromEnv() int {
	strNum := os.Getenv("TENANTS_NUMBER")
	if strNum == "" {
		fmt.Println("env var TENANTS_NUMBER was not set, defaulting to 1")
		return 1
	}
	num, err := strconv.Atoi(strNum)
	if err != nil {
		fmt.Println("error converting env var TENANTS_NUMBER to integer, defaulting to 1")
		return 1
	}
	return num
}

func getQuarkusPod(ctx *TestingContext) (*corev1.Pod, error) {
	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"app": quarkusAppName,
		}),
		k8sclient.InNamespace(quickStartNamespace),
	}

	quarkusPod := &corev1.PodList{}

	err := ctx.Client.List(goctx.TODO(), quarkusPod, listOptions...)
	if err != nil {
		return nil, fmt.Errorf("error listing quarkus pod: %v", err)
	}

	if len(quarkusPod.Items) == 0 {
		return nil, fmt.Errorf("quarkus pod doesn't exist in namespace: %v (err: %v)", quickStartNamespace, err)
	}

	return &quarkusPod.Items[0], nil
}

func createPortaClient(ctx *TestingContext, rhmi *integreatlyv1alpha1.RHMI, username string) (*portaclient.ThreeScaleClient, error) {
	// If username == "master", then function call is requesting the master portaClient which needs the master accessToken
	var accessToken string
	var admRoutePrefix string
	if username == "master" {
		// Get access token for portaClient
		s := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "system-seed",
				Namespace: ThreeScaleProductNamespace,
			},
		}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: s.Name, Namespace: s.Namespace}, s)
		if err != nil {
			return nil, fmt.Errorf("failed to get access token from secret %v during portaClient creation, error: %v", s, err)
		}
		accessToken = string(s.Data["MASTER_ACCESS_TOKEN"])
		admRoutePrefix = "master.apps"
	} else {
		// Get access token for portaClient
		s := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mt-signupaccount-3scale-access-token",
				Namespace: ThreeScaleProductNamespace,
			},
		}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: s.Name, Namespace: s.Namespace}, s)
		if err != nil {
			return nil, fmt.Errorf("failed to get access token from secret %v during portaClient creation, error: %v", s, err)
		}
		accessToken = string(s.Data[username])
		admRoutePrefix = fmt.Sprintf("%v-admin.apps", username)
	}

	// Create an admin portal for portaClient
	routes := &routev1.RouteList{}
	admRoute := routev1.Route{}
	found := false
	err := ctx.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{
		Namespace: ThreeScaleProductNamespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get 3scale route list during portaClient creation, error: %v", err)
	}
	for _, route := range routes.Items {
		if strings.Contains(route.Spec.Host, admRoutePrefix) {
			admRoute = route
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("failed to get 3scale route during portaClient creation, error: %v", err)
	}
	adminPortal, err := portaclient.NewAdminPortal("https", admRoute.Spec.Host, 443)
	if err != nil {
		return nil, fmt.Errorf("could not create admin portal during portaClient creation, error: %v", err)
	}

	/* #nosec */
	httpc := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   time.Second * 10,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: rhmi.Spec.SelfSignedCerts}, //#nosec G402 -- value is read from CR config
		},
	}

	// Create a new portaClient
	threescaleClient := portaclient.NewThreeScale(adminPortal, accessToken, httpc)
	if err != nil {
		return nil, fmt.Errorf("failed to create 3scale porta client, error: %v", err)
	}

	return threescaleClient, nil
}
