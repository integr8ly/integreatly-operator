// This script is automation of Test H24 procedure - to verify self-managed APIcast API gateway deployment
// Test does not include creation of custom policy
// Doc: https://access.redhat.com/documentation/en-us/red_hat_openshift_api_management/1/topic/a702e803-bbc8-47af-91a4-e73befd3da00
// This test case should prove that it is possible for customers to deploy self-managed APIcast

package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/rhmiConfigs"
	"golang.org/x/term"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"encoding/json"
	appsv1alpha1 "github.com/3scale/apicast-operator/apis/apps/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	testcommon "github.com/integr8ly/integreatly-operator/test/common"
	routev1 "github.com/openshift/api/route/v1"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	//"github.com/integr8ly/integreatly-operator/test/resources"
	"k8s.io/client-go/rest"
)

const (
	defaultDedicatedAdminName    = "customer-admin"
	adminPortalCredentialsSecret = "adminportal-credentials"
	accountOrgName               = "Developer"
	serviceSystemName            = "api"
	apiExampleApicast            = "apicast-example-apicast"
)

var (
	log                                                             = l.NewLogger()
	apicastNamespace, apicastImageStreamTag, apicastOperatorVersion string
	namespacePrefix                                                 string
	threeScaleNamespace                                             string
)

func main() {
	var useCustomerAdminUser, interactiveMode bool
	flag.StringVar(&apicastOperatorVersion, "apicast-operator-version", "", "APIcast Operator version")
	flag.StringVar(&apicastImageStreamTag, "apicast-image-stream-tag", "", "APIcast image stream tag")
	flag.StringVar(&apicastNamespace, "apicast-namespace", "selfmanaged-apicast", "Selfmanaged APIcast namespace")
	flag.BoolVar(&useCustomerAdminUser, "use-customer-admin-user", true, "Whether to use customer-admin user or not")
	flag.BoolVar(&interactiveMode, "interactive-mode", true, "Whether to run script in interactive mode or not")
	flag.StringVar(&namespacePrefix, "namespace-prefix", "redhat-rhoam-", "Namespace prefix of RHOAM. Defaults to redhat-rhoam-")
	flag.Parse()
	threeScaleNamespace = fmt.Sprintf("%s3scale", namespacePrefix)

	scheme := runtime.NewScheme()
	utilruntime.Must(operatorsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(appsv1alpha1.SchemeBuilder.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	client, err := k8sclient.New(config, k8sclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error("Error get Client k8sclient. Exit...", err)
		return
	}
	ctx := context.Background()

	err = cleanUpBeforeTest(ctx, client)
	if err != nil {
		log.Error("Error clean before test", err)
		return
	}

	// Create apicast namespace
	err = createNamespace(ctx, client)
	if err != nil {
		log.Error("Error create namespace "+apicastNamespace+". Exit...", err)
		return
	}

	// Get 3scale admin token
	token3scale, err := getThreeScaleAdminToken(ctx, client)
	if err != nil {
		log.Error("Error get 3scale admin Token. Exit...", err)
		return
	}

	// Customer login
	if useCustomerAdminUser {
		err = customerLogin(interactiveMode)
		if err != nil {
			log.Error("Error in customer login. Exit...", err)
			return
		}
	}

	// Import APIcast Image
	err = importApicastImage()
	if err != nil {
		log.Error("Error importApicastImage(), Exit...", err)
		return
	}

	// Create an adminportal-credentials secret
	threeScaleAdminPortal, err := getThreeScaleAdminPortal(ctx, client)
	if err != nil {
		log.Error("Error get 3scale admin portal. Exit...", err)
		return
	}
	err = createAdminPortalCredentialsSecret(ctx, client, token3scale, threeScaleAdminPortal)
	if err != nil {
		log.Error("Error create adminportal-credentials secret. Exit...", err)
		return
	}

	// Use self-managed APIcast instead of the builded one for API
	userKey, err := promoteSelfManagedAPIcast(ctx, client, interactiveMode, threeScaleAdminPortal, token3scale)
	if err != nil {
		log.Error("Error promote Self-managed APIcast. Exit...", err)
		return
	}

	// Install 3scale APIcast gateway operator
	err = installThreeScaleApicastGatewayOperator(client)
	if err != nil {
		log.Error("Error install 3scale APIcast gateway Operator. Exit...", err)
		return
	}

	// Create a self-managed APIcast
	err = createSelfManagedApicast(ctx, client)
	if err != nil {
		log.Error("Error create self-managed APIcast. Exit...", err)
		return
	}
	err = waitApiCastDeploymentReady(config)
	if err != nil {
		log.Error("self-managed APIcast deployment is not Ready ", err)
		return
	}
	// Create a route for the self-managed APIcast
	routeHost, err := createApicastRoute(ctx, client, threeScaleAdminPortal)
	if err != nil {
		log.Error("Error create route for self-managed APIcast. Exit...", err)
		return
	}

	// Validation of the Deployment
	res, err := validateDeployment(userKey, routeHost)
	if err != nil {
		log.Error("Validation of Self-managed APIcast deployment - Failed", err)
	} else {
		if res {
			log.Info("Validation of Self-managed APIcast deployment - Succeeded")
		} else {
			log.Error("Self-managed APIcast deployment issue", nil)
		}
	}

	log.Info("Self-managed APIcast deployment - Completed")

} //end of main()

func createNamespace(ctx context.Context, client k8sclient.Client) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: apicastNamespace,
		},
		Spec: corev1.NamespaceSpec{},
	}
	log.Info("Creating namespace " + ns.Name)
	err := client.Create(ctx, ns)
	if err != nil {
		log.Error("Error create namespace "+apicastNamespace, err)
		return err
	}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*3, false, func(ctx2 context.Context) (done bool, err error) {
		err = client.Get(ctx, k8sclient.ObjectKey{Name: ns.Name}, ns)
		if err != nil {
			log.Error("Error get namespace "+ns.Name, err)
			return false, err
		}
		if ns.Status.Phase != corev1.NamespaceActive {
			log.Info(ns.Name + "namespace status is not Active yet, waiting ...")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	//set namespace
	command := "oc project " + apicastNamespace
	err = runShellCommand(command)
	if err != nil {
		log.Error("Error set namespace "+ns.Name, err)
		return err
	}
	return nil
}

// Import APIcast Image
func importApicastImage() error {
	log.Info("Import APIcast Image")
	command := "oc import-image 3scale-amp2/apicast-gateway-rhel8:" + apicastImageStreamTag + " --from=registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:" + apicastImageStreamTag + " --confirm"
	err := runShellCommand(command)
	return err
}

// Get 3scale admin token
func getThreeScaleAdminToken(ctx context.Context, client k8sclient.Client) (string, error) {
	log.Info("Get 3scale admin Token")
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: s.Name, Namespace: threeScaleNamespace}, s)
	if err != nil {
		log.Error("Error get 3Scale admin token ", err)
		return "", err
	}
	accessToken := string(s.Data["ADMIN_ACCESS_TOKEN"])
	return accessToken, nil
}

func getThreeScaleAdminPortal(ctx context.Context, client k8sclient.Client) (string, error) {
	log.Info("Get 3scale admin Portal")
	opts := k8sclient.ListOptions{
		Namespace: threeScaleNamespace,
	}
	routes := routev1.RouteList{}
	err := client.List(ctx, &routes, &opts)
	if err != nil {
		log.Info("Error obtaining routes list in " + threeScaleNamespace + " namespace  ")
		return "", err
	}
	for _, route := range routes.Items {
		if strings.Contains(route.Spec.Host, "3scale-admin") {
			log.Info("Route: " + route.Name)
			annotationsMap := route.GetObjectMeta().GetAnnotations()
			adminPortal := annotationsMap["zync.3scale.net/host"]
			log.Info("3scale Admin portal: " + adminPortal)
			return adminPortal, nil
		}
	}
	return "", nil
}

func createAdminPortalCredentialsSecret(ctx context.Context, client k8sclient.Client,
	token3scale string, adminPortal string) error {
	log.Info("Create adminportal-credentials secret")
	adminPortalUrl := "https://" + token3scale + "@" + adminPortal
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminPortalCredentialsSecret,
			Namespace: apicastNamespace,
		},
		Data: map[string][]byte{},
	}
	secret.Data["AdminPortalURL"] = []byte(adminPortalUrl)
	err := client.Create(ctx, secret)
	if err != nil {
		log.Error("Error create adminportal-credentials Secret", err)
		return err
	}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*1, true, func(ctx2 context.Context) (bool, error) {
		err = client.Get(ctx, k8sclient.ObjectKey{Name: adminPortalCredentialsSecret, Namespace: apicastNamespace}, secret)
		if err != nil {
			if k8serr.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return nil
}

func deleteManagedApicastRoutes(ctx context.Context, client k8sclient.Client) error {
	log.Info("Delete api-3scale-apicast- routes in " + threeScaleNamespace + " namespace")
	routes := routev1.RouteList{}
	err := client.List(context.TODO(), &routes, &k8sclient.ListOptions{
		Namespace: threeScaleNamespace,
	})
	if err != nil {
		log.Error("Error obtaining routes list in "+threeScaleNamespace+" namespace  ", err)
		return err
	}
	for i := range routes.Items {
		route := routes.Items[i]
		if strings.Contains(route.Spec.Host, "api-3scale-apicast-") {
			err := client.Delete(ctx, &route)
			if err != nil {
				log.Error("error Delete Route", err)
				return err
			}
		}
	}
	return nil
}

// Install 3scale APIcast gateway operator
func installThreeScaleApicastGatewayOperator(client k8sclient.Client) error {
	log.Info("Install 3scale APIcast gateway Operator")
	verSplit := strings.Split(string(integreatlyv1alpha1.Version3Scale), ".")
	channelVer := verSplit[0] + "." + verSplit[1] // example: 2.11.0 -> 2.11
	targetNamespaces := []string{apicastNamespace}
	og := &v1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: apicastNamespace + "-",
			Namespace:    apicastNamespace,
			Generation:   1,
			Annotations: map[string]string{
				"olm.providedAPIs": "APIcast.v1alpha1.apps.3scale.net",
			},
		},
		Spec: v1.OperatorGroupSpec{
			TargetNamespaces: targetNamespaces,
		},
	}
	err := client.Create(context.TODO(), og)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		log.Error("Error create Operator Group in "+apicastNamespace+" namespace", err)
		return err
	}
	subscription := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apicast-operator",
			Namespace: apicastNamespace,
			Labels: map[string]string{
				"operators.coreos.com/apicast-operator.selfmanaged-apicast": "",
			},
			Generation: 1,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Channel:                "threescale-" + channelVer,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Package:                "apicast-operator",
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			StartingCSV:            "apicast-operator.v" + apicastOperatorVersion,
		},
	}
	err = client.Create(context.TODO(), subscription)
	if err != nil {
		log.Error("Error create Operator Subscription in "+apicastNamespace+" namespace", err)
		return err
	}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*5, false, func(ctx context.Context) (done bool, err error) {
		err = client.Get(context.TODO(), k8sclient.ObjectKey{Name: subscription.Name, Namespace: subscription.Namespace}, subscription)
		if err != nil {
			if k8serr.IsNotFound(err) {
				log.Info("Sunscription " + subscription.Name + "not created yet, waiting")
				return false, nil
			}
			return false, err
		}
		_, err = rhmiConfigs.GetLatestInstallPlan(context.TODO(), subscription, client)
		if err != nil {
			log.Info("Error get install plan for subscription " + subscription.Name + ", waiting")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Create a self-managed APIcast
func createSelfManagedApicast(ctx context.Context, client k8sclient.Client) error {
	log.Info("Create self-managed APIcast")
	specConfigurationLoadMode := "boot"
	specReplicas := int64(1)
	apicast := &appsv1alpha1.APIcast{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-apicast",
			Namespace: apicastNamespace,
			Labels: map[string]string{
				"operators.coreos.com/apicast-operator.selfmanaged-apicast": "",
			},
			Generation: 2,
			Annotations: map[string]string{
				"apicast.apps.3scale.net/operator-version": apicastOperatorVersion,
			},
			ManagedFields: []metav1.ManagedFieldsEntry{
				{APIVersion: "apps.3scale.net/v1alpha1"},
			},
		},
		Spec: appsv1alpha1.APIcastSpec{
			ConfigurationLoadMode: &specConfigurationLoadMode,
			//Image:                 &specImage,
			Replicas: &specReplicas,
			AdminPortalCredentialsRef: &corev1.LocalObjectReference{
				Name: adminPortalCredentialsSecret,
			},
		},
	}
	err := client.Create(ctx, apicast)
	if err != nil {
		log.Error("Error create APIcast - example-apicast", err)
		return err
	}
	return nil
}

// Create a route for the self-managed APIcast
func createApicastRoute(ctx context.Context, client k8sclient.Client, threeScaleAdminPortal string) (string, error) {
	log.Info("Create route for self-managed APIcast")
	hostsub := strings.TrimPrefix(threeScaleAdminPortal, "3scale-admin")
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apicast-example",
			Namespace: apicastNamespace,
			Labels: map[string]string{
				"app":                  "apicast",
				"threescale_component": "apicast",
			},
			Annotations: map[string]string{
				"openshift.io/host.generated": "true",
			},
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("proxy"),
			},
			TLS: &routev1.TLSConfig{
				Termination: "edge",
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: apiExampleApicast,
			},
			Host: "selfmanaged-staging" + hostsub,
		},
	}
	err := client.Create(ctx, route)
	if err != nil {
		log.Error("Error create Apicast Route", err)
		return "", err
	}
	err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*3, false, func(ctx2 context.Context) (done bool, err error) {
		err = client.Get(ctx, k8sclient.ObjectKey{Name: route.Name, Namespace: route.Namespace}, route)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	routeHost := route.Spec.Host
	log.Info("routeHost: " + routeHost)
	return routeHost, nil
}

func customerLogin(interactiveMode bool) error {
	log.Info("Customer Login")
	command := ""
	var err error
	if interactiveMode {
		message := "Copy Customer Admin user Token to command prompt (from openshift console -> copy login command Screen),\n"
		message += "and press Enter: "
		token, err := getCustomerAdminPasswordTokenFromTerm(message)
		if err != nil {
			log.Error("Error login customer, can't get token: ", err)
			return err
		}
		command = "oc login --token=" + token
	} else {
		customerAdminUsername := fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
		command = "oc login -u " + customerAdminUsername + " -p " + testcommon.TestingIdpPassword
	}
	err = runShellCommand(command)
	if err != nil {
		log.Error("Error login customer: ", err)
		return err
	}
	return nil
}

func getCustomerAdminPasswordTokenFromTerm(prompt string) (string, error) {
	fmt.Print(prompt)
	bytepw, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	pass := string(bytepw)
	return pass, nil
}

func runShellCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	return nil
}

// Configure the service to use the self-managed APIcast instead of the built-in APIcast for API.
// See https://access.redhat.com/documentation/en-us/red_hat_openshift_api_management/1/topic/a702e803-bbc8-47af-91a4-e73befd3da00
func promoteSelfManagedAPIcastInUI() (string, error) {
	message := "This is the manual step - Configure the service to use the self-managed APIcast instead of the built-in APIcast for API. \n"
	message += "a. Navigate to 3scale Admin Portal. You can use the following command to find route  \"oc get routes --namespace " + threeScaleNamespace + " | grep admin\" \n"
	message += "b. In the Products section, click API → Integration → Settings → APIcast Self Managed.\n"
	message += "c. Change the Staging Public Base URL. Replace api-3scale-apicast- with selfmanaged-. \n"
	message += "d. Click Update Product. \n"
	message += "e. Click API → Configuration → Promote to Staging and Promote to Production\n"
	message += "f. Copy user_key value (from Staging APIcast - Example curl for testing) to command prompt \n"
	log.Info(message)
	fmt.Print("Waiting for manual step completion. Copy user_key value here and Press enter when done :")
	userKey, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		log.Error("Error: ", err)
		return "", err
	}
	log.Info("Promote - completed, continue ...")
	return string(userKey[:]), err
}

func promoteSelfManagedAPIcast(ctx context.Context, client k8sclient.Client, interactiveMode bool,
	threeScaleAdminPortal string, token3scale string) (string, error) {
	userKey := ""
	var err error
	if interactiveMode {
		userKey, err = promoteSelfManagedAPIcastInUI()
		if err != nil {
			log.Error("Error promote Self-managed APIcast. Exit...", err)
			return "", err
		}
	} else {
		err = deleteManagedApicastRoutes(ctx, client)
		if err != nil {
			log.Error("Error deleting Managed Apicast Routes : ", err)
			return "", err
		}
		serviceId, err := getServiceId(threeScaleAdminPortal, token3scale)
		if err != nil {
			log.Error("Error get Service ID", err)
			return "", err
		}
		err = apiCastConfigPromote(threeScaleAdminPortal, token3scale, serviceId)
		if err != nil {
			log.Error("Error in 3scale api Proxy Config Promote : ", err)
			return "", err
		}
		userKey, err = getUserKey(threeScaleAdminPortal, token3scale, serviceId)
		if err != nil {
			log.Error("Error get user key: ", err)
			return "", err
		}
	}
	return userKey, err
}

func validateDeploymentRequest(userKey, routeHost string) (int, error) {
	query := make(url.Values)
	query.Add("user_key", userKey)
	httpRequest := &url.URL{
		Scheme:     "https",
		Host:       routeHost,
		ForceQuery: false,
		RawQuery:   query.Encode(),
	}
	resp, err := http.Get(httpRequest.String())
	if err != nil {
		log.Error("HTTP Get error", err)
		return 0, err
	}
	//log.Info("Response Code: " + strconv.Itoa(resp.StatusCode))
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		_, err := ioutil.ReadAll(resp.Body) //bytes,err:= ...
		if err != nil {
			log.Error("Unable to read response body: ", err)
			return 0, nil
		}
		return http.StatusOK, nil
	} else {
		fmt.Printf("Expected status %v but got %v\n", http.StatusOK, resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func validateDeployment(userKey, routeHost string) (bool, error) {
	log.Info("Validation of deployment")
	responseCode := 0
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*3, false, func(ctx context.Context) (done bool, err error) {
		responseCode, err = validateDeploymentRequest(userKey, routeHost)
		if err != nil {
			return false, err
		}
		if responseCode != http.StatusOK {
			return false, nil
		}
		return true, nil
	})
	if err != nil && responseCode != http.StatusOK {
		return false, err
	}
	fmt.Printf("Response Code: %v\n", responseCode)
	return true, nil
}

func apiCastConfigPromote(threeScaleAdminPortal string, token3scale string, serviceId string) error {
	threeScaleAdminPortalServiceUrl := "https://" + threeScaleAdminPortal + "/admin/api/services/" + serviceId
	// Switch to self_managed Apicast
	err := serviceUpdate(threeScaleAdminPortalServiceUrl, token3scale)
	if err != nil {
		log.Error("Error in Service Update - switch to self-managed deployment", err)
		return err
	}
	// Set Staging and Production Public Base URL
	err = proxyUpdate(threeScaleAdminPortal, threeScaleAdminPortalServiceUrl, token3scale)
	if err != nil {
		log.Error("Error in Proxy Update - set public base URL for staging and production", err)
		return err
	}
	// Promotes the APIcast configuration to the Staging Environment
	err = proxyDeploy(threeScaleAdminPortalServiceUrl, token3scale)
	if err != nil {
		log.Error("Error in Proxy Deploy - promotes the APIcast configuration to the staging environment", err)
		return err
	}
	// Promotes a Proxy Config from Staging environment to Production environment.
	err = proxyConfigPromote(threeScaleAdminPortalServiceUrl, token3scale)
	if err != nil {
		log.Error("Error in Proxy Config Promote - promotes the APIcast configuration to the production environment", err)
		return err
	}
	return nil
}

// Switch to self_managed Apicast
// 3scale API: Service Update
func serviceUpdate(threeScaleAdminPortalServiceUrl string, token3scale string) error {
	url := threeScaleAdminPortalServiceUrl + ".xml"
	log.Info("Service Update URL: " + url)
	data, err := json.Marshal(map[string]string{
		"access_token":      token3scale,
		"deployment_option": "self_managed",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status %v but got %v", http.StatusOK, resp.StatusCode)
	}
	return nil
}

// Set Staging and Production Public Base URL
// 3scale API: Proxy Update
func proxyUpdate(threeScaleAdminPortal, threeScaleAdminPortalServiceUrl string, token3scale string) error {
	url := threeScaleAdminPortalServiceUrl + "/proxy.xml"
	log.Info("Proxy Update URL: " + url)
	hostsub := strings.TrimPrefix(threeScaleAdminPortal, "3scale-admin")
	productionEndPoint := "https://api-3scale-apicast-production" + hostsub
	stagingEndPoint := "https://selfmanaged-staging" + hostsub
	data, err := json.Marshal(map[string]string{
		"access_token":     token3scale,
		"endpoint":         productionEndPoint,
		"sandbox_endpoint": stagingEndPoint,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	client := &http.Client{}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status %v or %v but got %v", http.StatusCreated, http.StatusOK, resp.StatusCode)
	}
	return nil
}

// Promotes the APIcast configuration to the Staging Environment
// 3scale API: Proxy Deploy
func proxyDeploy(threeScaleAdminPortalServiceUrl string, token3scale string) error {
	url := threeScaleAdminPortalServiceUrl + "/proxy/deploy.xml"
	log.Info("Proxy Deploy URL: " + url)
	data, err := json.Marshal(map[string]string{
		"access_token": token3scale,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected status %v but got %v", http.StatusCreated, resp.StatusCode)
	}
	return nil
}

// Promotes a Proxy Config from Staging environment to Production environment.
// 3scale API: Proxy Config Promote
func proxyConfigPromote(threeScaleAdminPortalServiceUrl string, token3scale string) error {
	latestVer, err := getProxyConfigLatestVersion(threeScaleAdminPortalServiceUrl, token3scale)
	if err != nil {
		return err
	}
	url := threeScaleAdminPortalServiceUrl + "/proxy/configs/sandbox/" + latestVer + "/promote.json"
	log.Info("Proxy Config Promote URL: " + url)
	data, err := json.Marshal(map[string]string{
		"access_token": token3scale,
		"to":           "production",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("can't promote to Production. Expected status %v but got %v\n", http.StatusCreated, resp.StatusCode)
	}
	return nil
}

func checkApicastNamespaceExists(ctx context.Context, serverClient k8sclient.Client) (bool, error) {
	namespaceList := &corev1.NamespaceList{}
	err := serverClient.List(ctx, namespaceList)
	if err != nil {
		return false, err
	}
	for _, namespace := range namespaceList.Items {
		if namespace.Name == apicastNamespace {
			return true, nil
		}
	}
	return false, nil
}

func cleanUpBeforeTest(ctx context.Context, serverClient k8sclient.Client) error {
	apiCastNsExists, err := checkApicastNamespaceExists(ctx, serverClient)
	if err != nil {
		return err
	}
	if apiCastNsExists {
		command := "oc delete project " + apicastNamespace
		err = runShellCommand(command)
		if err != nil {
			return err
		}
		//wait for deletion of apicastNamespace
		err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*5, false, func(ctx2 context.Context) (done bool, err error) {
			apiCastNsExists, err = checkApicastNamespaceExists(ctx, serverClient)
			if err != nil {
				return false, err
			}
			if apiCastNsExists {
				log.Info(apicastNamespace + " namespace not deleted yet, waiting")
				return false, nil
			}
			log.Info(apicastNamespace + " namespace deleted")
			return true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getServiceId(threeScaleAdminPortal string, token3scale string) (string, error) {
	url := "https://" + threeScaleAdminPortal + "/admin/api/services.xml"
	bytesresp, err := getRequest(token3scale, url)
	if err != nil {
		return "", err
	}
	type Service struct {
		Id         string `xml:"id"`
		SystemName string `xml:"system_name"`
		Body       string `xml:",chardata"`
	}
	type Services struct {
		XMLName  xml.Name  `xml:"services"`
		Services []Service `xml:"service"`
	}
	var svc Services
	err = xml.Unmarshal(bytesresp, &svc)
	if err != nil {
		return "", fmt.Errorf("unable to Unmarshal get services response: %s", err)
	}
	svcId := ""
	for i := 0; i < len(svc.Services); i++ {
		if svc.Services[i].SystemName == serviceSystemName {
			svcId = svc.Services[i].Id
			break
		}
	}
	if svcId == "" {
		return svcId, fmt.Errorf("unable to parse get services response xml")
	}
	log.Info("Service ID found: " + svcId)
	return svcId, nil
}

func getAccountId(threeScaleAdminPortal string, token3scale string) (string, error) {
	url := "https://" + threeScaleAdminPortal + "/admin/api/accounts.xml"
	bytesresp, err := getRequest(token3scale, url)
	if err != nil {
		return "", err
	}
	type Account struct {
		Id      string `xml:"id"`
		OrgName string `xml:"org_name"`
		Body    string `xml:",chardata"`
	}
	type Accounts struct {
		XMLName  xml.Name  `xml:"accounts"`
		Accounts []Account `xml:"account"`
	}
	acc := Accounts{}
	err = xml.Unmarshal(bytesresp, &acc)
	if err != nil {
		return "", fmt.Errorf("unable to Unmarshal get accounts response: %s", err)
	}
	accId := ""
	for i := 0; i < len(acc.Accounts); i++ {
		if acc.Accounts[i].OrgName == accountOrgName {
			accId = acc.Accounts[i].Id
			break
		}
	}
	if accId == "" {
		return accId, fmt.Errorf("unable to parse get accounts response xml")
	}
	log.Info("Account ID found: " + accId)
	return accId, nil
}

func getUserKey(threeScaleAdminPortal string, token3scale string, serviceId string) (string, error) {
	accountId, err := getAccountId(threeScaleAdminPortal, token3scale)
	if err != nil {
		return "", err
	}
	url := "https://" + threeScaleAdminPortal + "/admin/api/accounts/" + accountId + "/applications.xml"
	bytesresp, err := getRequest(token3scale, url)
	if err != nil {
		return "", err
	}
	type Application struct {
		Id        string `xml:"id"`
		ServiceId string `xml:"service_id"`
		UserKey   string `xml:"user_key"`
		Body      string `xml:",chardata"`
	}
	type Applications struct {
		XMLName      xml.Name      `xml:"applications"`
		Applications []Application `xml:"application"`
	}
	app := Applications{}
	err = xml.Unmarshal(bytesresp, &app)
	if err != nil {
		return "", fmt.Errorf("unable to Unmarshal get applications response: %s", err)
	}
	userKey := ""
	for i := 0; i < len(app.Applications); i++ {
		if app.Applications[i].ServiceId == serviceId {
			userKey = app.Applications[i].UserKey
			break
		}
	}
	if userKey == "" {
		return userKey, fmt.Errorf("unable to parse get applications response xml")
	}
	log.Info("userKey found")
	return userKey, nil
}

func getProxyConfigLatestVersion(threeScaleAdminPortalServiceUrl string, token3scale string) (string, error) {
	url := threeScaleAdminPortalServiceUrl + "/proxy/configs/sandbox/latest.json"
	log.Info("get Proxy Config Latest Version url: " + url)
	bytesresp, err := getRequest(token3scale, url)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	err = json.Unmarshal(bytesresp, &result)
	if err != nil {
		return "", err
	}
	proxyConfig := result["proxy_config"].(map[string]interface{})
	versionNum := ""
	for key, value := range proxyConfig {
		if key == "version" {
			versionNum = strconv.Itoa(int(value.(float64)))
			break
		}
	}
	log.Info("Latest Proxy Config version: " + versionNum)
	return versionNum, nil
}

func getRequest(token3scale string, url string) ([]byte, error) {
	log.Info("sendGetRequest, url: " + url)
	data, err := json.Marshal(map[string]string{
		"access_token": token3scale,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status %v but got %v", http.StatusOK, resp.StatusCode)
	}
	bytesresp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return bytesresp, nil
}

func waitApiCastDeploymentReady(config *rest.Config) error {
	log.Info("Wait APIcast deployment ready")
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*3, false, func(ctx context.Context) (done bool, err error) {
		testingContext, err := testcommon.NewTestingContext(config)
		if err != nil || testingContext == nil {
			log.Error("failed to create testing context", err)
			return false, err
		}
		deployment, err := testingContext.KubeClient.AppsV1().Deployments(apicastNamespace).Get(context.TODO(), apiExampleApicast, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		if int(deployment.Status.ReadyReplicas) >= 1 {
			log.Info("Replicas Ready: " + strconv.Itoa(int(deployment.Status.ReadyReplicas)))
			return true, nil
		}
		return false, err
	})
	if err != nil {
		return fmt.Errorf("APIcast deployment %v is not ready: %s", apiExampleApicast, err)
	}
	return nil
}
