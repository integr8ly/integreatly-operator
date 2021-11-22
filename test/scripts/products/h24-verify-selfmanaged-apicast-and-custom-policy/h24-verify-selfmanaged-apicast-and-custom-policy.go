// This script is automation of Test H24 procedure - to verify self-managed APIcast API gateway deployment
// Test does not include creation of custom policy
// Procedure: ./test-cases/tests/products/h24-verify-selfmanaged-apicast-and-custom-policy.md
// Doc: https://access.redhat.com/documentation/en-us/red_hat_openshift_api_management/1/topic/a702e803-bbc8-47af-91a4-e73befd3da00
// This test case should prove that it is possible for customers to deploy self-managed APIcast

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	appsv1alpha1 "github.com/3scale/apicast-operator/pkg/apis/apps/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	routev1 "github.com/openshift/api/route/v1"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log                                                             = l.NewLogger()
	apicastNamespace, apicastImageStreamTag, apicastOperatorVersion string
	namespacePrefix                                                 string
	threeScaleNamespace                                             string
)

func main() {
	var createTestingIdp, useCustomerAdminUser, promoteManually bool
	flag.StringVar(&apicastOperatorVersion, "apicast-operator-version", "", "APIcast Operator version")
	flag.StringVar(&apicastImageStreamTag, "apicast-image-stream-tag", "", "APIcast image stream tag")
	flag.StringVar(&apicastNamespace, "apicast-namespace", "selfmanaged-apicast", "Selfmanaged APIcast namespace")
	flag.BoolVar(&createTestingIdp, "create-testing-idp", true, "Whether to create testing-idp or not")
	flag.BoolVar(&useCustomerAdminUser, "use-customer-admin-user", true, "Whether to use customer-admin user or not")
	flag.BoolVar(&promoteManually, "promote-manually", true, "Whether to do Promote manually or not")
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

	// Create apicast namespace
	err = createNamespace(ctx, client)
	if err != nil {
		log.Error("Error create namespace "+apicastNamespace+". Exit...", err)
		return
	}
	time.Sleep(5 * time.Second)

	// Get 3scale admin token
	// will be used later to create Admin Portal Credentials Secret
	token3scale, err := getThreeScaleAdminToken(ctx, client)
	if err != nil {
		log.Error("Error get 3scale admin Token. Exit...", err)
		return
	}

	// Create customer users in dedicated-admins group
	if createTestingIdp {
		customerAdminPassword, err := getCustomerAdminPasswordTokenFromTerm("Enter Customer Admin user Password :")
		if err != nil {
			log.Error("Error get Customer Admin Password. Exit...", err)
			return
		}
		err = createCustomerUsers(customerAdminPassword)
		if err != nil {
			log.Error("Error create Customer Users. Exit...", err)
			return
		}
	} else {
		log.Info("Skipped creation of testing-idp. To enable it - update test.sh - set -create-testing-idp=\"yes\" ")
	}

	// Customer login
	if useCustomerAdminUser {
		err = customerLogin()
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
	time.Sleep(5 * time.Second)

	// Use self-managed APIcast instead of the builded one for API
	userKey, err := promoteSelfManagedAPIcast(ctx, client, promoteManually)
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
	time.Sleep(120 * time.Second) //TODO- recheck if sleep needed

	// Create a self-managed APIcast
	err = createSelfManagedApicast(ctx, client)
	if err != nil {
		log.Error("Error create self-managed APIcast. Exit...", err)
		return
	}

	// Create a route for the self-managed APIcast
	routeHost, err := createApicastRoute(ctx, client, threeScaleAdminPortal)
	if err != nil {
		log.Error("Error create route for self-managed APIcast. Exit...", err)
		return
	}
	time.Sleep(60 * time.Second)

	// Validation of the Deployment
	res, err := validateDeployment(userKey, routeHost)
	if err != nil {
		log.Error("Validation of Self-managed APIcast API gateway Deployed - Failed", err)
	} else {
		if res {
			log.Info("Validation of Self-managed APIcast API gateway Deployment - Succeeded")
		} else {
			log.Error("Self-managed APIcast API gateway Deployment issue", nil)
		}
	}

	log.Info("Self-managed APIcast API gateway - Deployment script Completed")

} //end of main()

// Create customer users in dedicated-admins group
func createCustomerUsers(customerAdminPassword string) error {
	log.Info("Creating customer users in dedicated-admins group")
	command := "PASSWORD=" + customerAdminPassword + " DEDICATED_ADMIN_PASSWORD=" + customerAdminPassword + " ../../../../scripts/setup-sso-idp.sh"
	err := run_shell_command(command)
	return err
}

func createNamespace(ctx context.Context, client k8sclient.Client) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: apicastNamespace,
		},
		Spec: corev1.NamespaceSpec{},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		log.Info("Creating namespace " + ns.Name)
		err = client.Create(ctx, ns)
		if err != nil {
			log.Error("Error create namespace "+apicastNamespace, err)
			return err
		}
	} else {
		log.Info("Namespace " + apicastNamespace + " already exists")
	}
	err = client.Get(ctx, k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		log.Error("Error get namespace"+ns.Name, err)
		return err
	}
	command := "oc project " + apicastNamespace
	return run_shell_command(command)
}

// Import APIcast Image
func importApicastImage() error {
	log.Info("Import APIcast Image")
	command := "oc import-image 3scale-amp2/apicast-gateway-rhel8:" + apicastImageStreamTag + " --from=registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:" + apicastImageStreamTag + " --confirm"
	err := run_shell_command(command)
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
	token3scale string, admin_portal string) error {
	log.Info("Create adminportal-credentials secret")
	adminPortalUrl := "https://" + token3scale + "@" + admin_portal
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "adminportal-credentials",
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
	return nil
}

// Use self-managed APIcast instead of the builded one for API
// This is attempt to automate the Promotion steps in 3scale Admin Portal
// Require more investigation. Recommended to use manual meanwhile
func deleteManagedApicastRoutes(ctx context.Context, client k8sclient.Client) error {
	log.Info("Simulate Promotion in 3scale Admin Portal - Delete api-3scale-apicast- routes in " + threeScaleNamespace + " namespace")
	routes := routev1.RouteList{}
	err := client.List(context.TODO(), &routes, &k8sclient.ListOptions{
		Namespace: threeScaleNamespace,
	})
	if err != nil {
		log.Error("Error obtaining routes list in "+threeScaleNamespace+" namespace  ", err)
		return err
	}
	for _, route := range routes.Items {
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
				Name: "adminportal-credentials",
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
				Name: "apicast-example-apicast",
			},
			Host: "selfmanaged-staging" + hostsub,
		},
	}
	err := client.Create(ctx, route)
	if err != nil {
		log.Error("Error create Apicast Route", err)
		return "", err
	}
	routeHost := route.Spec.Host
	log.Info("routeHost: " + routeHost)
	return routeHost, nil
}

func customerLogin() error {
	log.Info("Customer Login")
	message := "Copy Customer Admin user Token to command prompt (from openshift console -> copy login command Screen),\n"
	message += "and press Enter: "
	token, err := getCustomerAdminPasswordTokenFromTerm(message)
	if err != nil {
		log.Error("Error login customer, can't get token: ", err)
		return err
	}
	command := "oc login --token=" + token
	err = run_shell_command(command)
	if err != nil {
		log.Error("Error login customer: ", err)
		return err
	}
	return nil
}

func getCustomerAdminPasswordTokenFromTerm(prompt string) (string, error) {
	fmt.Print(prompt)
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	pass := string(bytepw)
	return pass, nil
}

func run_shell_command(command string) error {
	//log.Info(command)
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
// step 11
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
	userKey, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Error("Error: ", err)
		return "", err
	}
	log.Info("Promote - completed, continue ...")
	return string(userKey[:]), err
}

func promoteSelfManagedAPIcast(ctx context.Context, client k8sclient.Client, promoteManually bool) (string, error) {
	userKey := ""
	var err error
	if promoteManually {
		userKey, err = promoteSelfManagedAPIcastInUI()
		if err != nil {
			log.Error("Error promote Self-managed APIcast. Exit...", err)
			return "", err
		}
	} else {
		//TODO - will be done in full automation task, https://issues.redhat.com/browse/MGDAPI-3037
		deleteManagedApicastRoutes(ctx, client)
		userKey, err = getUserKey()
		if err != nil {
			log.Error("Error: ", err)
			return "TODO. Not implemented. Please use option \"promote-manually=true\" ", err
		}
	}
	return userKey, err
}

func getUserKey() (string, error) {
	//TODO - will be done in full automation task, https://issues.redhat.com/browse/MGDAPI-3037
	return "TODO", nil
}

func validateDeploymentRequest(userKey, routeHost string) (int, error) {
	httpRequest := "https://" + routeHost + "/?user_key=" + userKey
	//log.Info("HTTP Request: " + httpRequest)
	resp, err := http.Get(httpRequest)
	if err != nil {
		log.Error("HTTP Get error", err)
		return 0, err
	}
	log.Info("Response Code: " + strconv.Itoa(resp.StatusCode))
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		_, err := ioutil.ReadAll(resp.Body) //bytes,err:= ...
		if err != nil {
			log.Error("unable to read response body: ", err)
			return 0, nil
		}
		//log.Info("Response Body: " + string(bytes))
		return http.StatusOK, nil
	}
	return resp.StatusCode, fmt.Errorf("expected status %v but got %v", http.StatusOK, resp.StatusCode)
}

func validateDeployment(userKey, routeHost string) (bool, error) {
	responseCode := 0
	err := retry(10, 20*time.Second, func() (err error) {
		responseCode, err = validateDeploymentRequest(userKey, routeHost)
		return
	})
	if err != nil && responseCode != http.StatusOK {
		return false, err
	}
	return true, nil
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(sleep)
		log.Info("retrying ...")
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
