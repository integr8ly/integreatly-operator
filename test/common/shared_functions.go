package common

import (
	"bytes"
	goctx "context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	pkgresources "github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/openshift/api/project/v1"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	routev1 "github.com/openshift/api/route/v1"
	"golang.org/x/net/publicsuffix"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/remotecommand"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTestingContext(kubeConfig *rest.Config) (*TestingContext, error) {
	kubeclient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the kubeclient: %v", err)
	}

	apiextensions, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the apiextension client: %v", err)
	}

	if err := extscheme.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add api extensions scheme to runtime scheme: (%v)", err)
	}
	if err := configv1.Install(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add route scheme to runtime scheme: (%v)", err)
	}
	if err := rhmiv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add integreatly scheme to runtime scheme: (%v)", err)
	}
	if err := rhmiv1alpha1.AddToSchemes.AddToScheme(scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add integreatly scheme to runtime scheme: (%v)", err)
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()

	dynClient, err := k8sclient.New(kubeConfig, k8sclient.Options{Scheme: scheme.Scheme, Mapper: restMapper})
	if err != nil {
		return nil, fmt.Errorf("failed to build the dynamic client: %v", err)
	}

	urlToCheck := kubeConfig.Host
	consoleUrl, err := getConsoleRoute(dynClient)
	if err != nil {
		return nil, err
	}
	if consoleUrl != nil {
		// use the console url if we can as when the tests are executed inside a pod, the kubeConfig.Host value is the ip address of the pod
		urlToCheck = *consoleUrl
	}

	selfSignedCerts, err := HasSelfSignedCerts(fmt.Sprintf("https://%s", urlToCheck), http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("failed to determine self-signed certs status on cluster: %w", err)
	}

	httpClient, err := NewTestingHTTPClient(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create testing http client: %v", err)
	}

	return &TestingContext{
		Client:          dynClient,
		KubeConfig:      kubeConfig,
		KubeClient:      kubeclient,
		ExtensionClient: apiextensions,
		HttpClient:      httpClient,
		SelfSignedCerts: selfSignedCerts,
	}, nil
}

func NewTestingHTTPClient(kubeConfig *rest.Config) (*http.Client, error) {
	selfSignedCerts, err := HasSelfSignedCerts(kubeConfig.Host, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("failed to determine self-signed certs status on cluster: %w", err)
	}

	// Create the http client with a cookie jar
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create new cookie jar: %v", err)
	}

	/* #nosec */
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: selfSignedCerts}, // gosec G402, Used only in tests, cluster checked for self-signed certs
	}

	if selfSignedCerts {
		fmt.Println("TLS insecure skip verify is enabled")
	}

	httpClient := &http.Client{
		Jar:           jar,
		Transport:     transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return nil },
	}

	return httpClient, nil
}

func HasSelfSignedCerts(url string, httpClient *http.Client) (bool, error) {
	if _, err := httpClient.Get(url); err != nil {
		if _, ok := errors.Unwrap(err).(x509.UnknownAuthorityError); !ok {
			return false, fmt.Errorf("error while performing self-signed certs test request: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func getConsoleRoute(client k8sclient.Client) (*string, error) {
	route := &routev1.Route{}
	if err := client.Get(goctx.TODO(), types.NamespacedName{Name: OpenShiftConsoleRoute, Namespace: OpenShiftConsoleNamespace}, route); err != nil {
		return nil, err
	}
	if len(route.Status.Ingress) > 0 {
		return &route.Status.Ingress[0].Host, nil
	}
	return nil, nil
}

func GetInstallType(config *rest.Config) (string, error) {

	testingContext, err := NewTestingContext(config)
	if err != nil {
		return "", fmt.Errorf("failed to create testing context %s", err)
	}
	rhmi, err := GetRHMI(testingContext.Client, true)

	if err != nil {
		return "", err
	}

	return rhmi.Spec.Type, nil
}

func GetRHMI(client k8sclient.Client, failNotExist bool) (*rhmiv1alpha1.RHMI, error) {
	installationList := &rhmiv1alpha1.RHMIList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(RHOAMOperatorNamespace),
	}
	err := client.List(goctx.TODO(), installationList, listOpts...)
	if err != nil {
		return nil, err
	}
	if len(installationList.Items) == 0 && failNotExist == true {
		return nil, fmt.Errorf("rhmi CRs does not exist: %v namespace: '%v', list: %v", err, RHOAMOperatorNamespace, installationList)
	}
	if len(installationList.Items) == 0 && failNotExist == false {
		return nil, nil
	}
	if len(installationList.Items) != 1 {
		return nil, fmt.Errorf("unexpected number of rhmi CRs: %w", err)
	}
	return &installationList.Items[0], nil
}

func GetHappyPathTestCases(installType string) []TestCase {
	testCases := []TestCase{}
	for _, testSuite := range HAPPY_PATH_TESTS {
		for _, tsInstallType := range testSuite.InstallType {
			if string(tsInstallType) == installType {
				testCases = append(testCases, testSuite.TestCases...)
			}
		}
	}

	return testCases
}

func getTimeStampPrefix() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d_%02d_%02dT%02d_%02d_%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}

func ExecToPodArgs(client kubernetes.Interface, config *rest.Config, command []string, podName, namespace, container string) (string, error) {
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	newScheme := runtime.NewScheme()
	if err := corev1.AddToScheme(newScheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(newScheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("error in Stream: %v (%s)", err, stderr.String())
	}
	return stdout.String(), nil
}

func ExecToPod(client kubernetes.Interface, config *rest.Config, command, podName, namespace, container string) (string, error) {
	return ExecToPodArgs(client, config, strings.Fields(command), podName, namespace, container)
}

func execToPod(command string, podName string, namespace string, container string, ctx *TestingContext) (string, error) {
	return ExecToPod(ctx.KubeClient, ctx.KubeConfig, command, podName, namespace, container)
}

// Is the cluster using on cluster or external storage
func isClusterStorage(ctx *TestingContext) (bool, error) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		return true, fmt.Errorf("error getting RHMI CR: %v", err)
	}

	if rhmi.Spec.UseClusterStorage == "true" {
		return true, nil
	}
	return false, nil
}

// Common function to perform CRUDL and verifying their expected permissions
func verifyCRUDLPermissions(t TestingTB, openshiftClient *resources.OpenshiftClient, expectedPermission ExpectedPermissions) {
	// Perform LIST Request
	resp, err := openshiftClient.DoOpenshiftGetRequest(expectedPermission.ListPath)

	if err != nil {
		t.Errorf("failed to perform LIST request with error : %s", err)
	}

	if resp.StatusCode != expectedPermission.ExpectedListStatusCode {
		t.Errorf("unexpected response from LIST request, expected %d status but got: %v", expectedPermission.ExpectedListStatusCode, resp)
	}

	// Perform CREATE Request
	bodyBytes, err := json.Marshal(expectedPermission.ObjectToCreate)

	if err != nil {
		t.Errorf("failed to marshal object to json for create request: %s", err)
	}

	resp, err = openshiftClient.DoOpenshiftPostRequest(expectedPermission.ListPath, bodyBytes)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Log("Responses body close error: ", err)
		}
	}(resp.Body)
	if err != nil {
		t.Errorf("failed to perform CREATE request with error : %s", err)
	}

	if resp.StatusCode != expectedPermission.ExpectedCreateStatusCode {
		t.Errorf("unexpected response from CREATE request, expected %d status but got: %v", expectedPermission.ExpectedCreateStatusCode, resp)
	}

	// Perform GET Request
	resp, err = openshiftClient.DoOpenshiftGetRequest(expectedPermission.GetPath)

	if err != nil {
		t.Errorf("failed to perform GET request with error : %s", err)
	}

	if resp.StatusCode != expectedPermission.ExpectedReadStatusCode {
		t.Errorf("unexpected response from GET request, expected %d status but got: %v", expectedPermission.ExpectedReadStatusCode, resp)
	}

	// Perform UPDATE Request
	bodyBytes, err = ioutil.ReadAll(resp.Body) // Use response from GET

	if err != nil {
		t.Errorf("failed to read response body for update request: %s", err)
	}

	resp, err = openshiftClient.DoOpenshiftPutRequest(expectedPermission.GetPath, bodyBytes)

	if err != nil {
		t.Errorf("failed to perform UPDATE request with error : %s", err)
	}

	if resp.StatusCode != expectedPermission.ExpectedUpdateStatusCode {
		t.Errorf("unexpected response from UPDATE request, expected %d status but got: %v", expectedPermission.ExpectedUpdateStatusCode, resp)
	}

	// Perform DELETE Request
	resp, err = openshiftClient.DoOpenshiftDeleteRequest(expectedPermission.GetPath)

	if err != nil {
		t.Errorf("failed to perform DELETE request with error : %s", err)
	}

	if resp.StatusCode != expectedPermission.ExpectedDeleteStatusCode {
		t.Errorf("unexpected response from DELETE request, expected %d status but got: %v", expectedPermission.ExpectedDeleteStatusCode, resp)
	}

	// Close the response body
	err = resp.Body.Close()
	if err != nil {
		t.Errorf("failed to close response body: %s", err)
	}
}

func GetIDPBasedTestCases(installType string) []TestCase {
	testCases := []TestCase{}
	for _, testSuite := range IDP_BASED_TESTS {
		for _, tsInstallType := range testSuite.InstallType {
			if string(tsInstallType) == installType {
				testCases = append(testCases, testSuite.TestCases...)
			}
		}
	}
	return testCases
}

func GetAllTestCases(installType string) []TestCase {
	var testCases []TestCase
	for _, testSuite := range ALL_TESTS {
		for _, tsInstallType := range testSuite.InstallType {
			if string(tsInstallType) == installType {
				testCases = append(testCases, testSuite.TestCases...)
			}
		}
	}
	return testCases
}

func GetScalabilityTestCases(installType string) []TestCase {
	var testCases []TestCase
	for _, testSuite := range SCALABILITY_TESTS {
		for _, tsInstallType := range testSuite.InstallType {
			if string(tsInstallType) == installType {
				testCases = append(testCases, testSuite.TestCases...)
			}
		}
	}
	return testCases
}

func GetGCPTestCases(installType string) []TestCase {
	var testCases []TestCase
	for _, testSuite := range GCP_TESTS {
		for _, tsInstallType := range testSuite.InstallType {
			if string(tsInstallType) == installType {
				testCases = append(testCases, testSuite.TestCases...)
			}
		}
	}
	return testCases
}

func GetAWSSpecificTestCases(installType string) []TestCase {
	var testCases []TestCase
	for _, testSuite := range AWS_SPECIFIC_TESTS {
		for _, tsInstallType := range testSuite.InstallType {
			if string(tsInstallType) == installType {
				testCases = append(testCases, testSuite.TestCases...)
			}
		}
	}
	return testCases
}

//func GetClusterScopedTestCases(installType string) []TestCase {
//	testCases := []TestCase{}
//	for _, testSuite := range THREESCALE_CLUSTER_SCOPED_TESTS {
//		for _, tsInstallType := range testSuite.InstallType {
//			if string(tsInstallType) == installType {
//				testCases = append(testCases, testSuite.TestCases...)
//			}
//		}
//	}
//	return testCases
//}

//func IsClusterScoped(restConfig *rest.Config) (bool, error) {
//	newTestingContext, err := NewTestingContext(restConfig)
//	if err != nil {
//		return false, err
//	}
//	threeScaleOperatorGroup := &operatorsv1.OperatorGroup{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "rhmi-registry-og",
//			Namespace: ThreeScaleOperatorNamespace,
//		},
//	}
//
//	err = newTestingContext.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: "rhmi-registry-og", Namespace: ThreeScaleOperatorNamespace}, threeScaleOperatorGroup)
//	if err != nil {
//		return false, err
//	}
//
//	for _, namespace := range threeScaleOperatorGroup.Status.Namespaces {
//		if namespace == "" {
//			return true, nil
//		}
//	}
//
//	return false, nil
//}

func writeObjToYAMLFile(obj interface{}, out string) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(out, data, 0600)
}

func WriteRHMICRToFile(client dynclient.Client, file string) error {
	rhmi, err := GetRHMI(client, true)
	if err != nil {
		return fmt.Errorf("failed to write RHMI cr due to error %w", err)
	}
	return writeObjToYAMLFile(rhmi, file)
}

func WaitForRHMIStageToComplete(t ginkgo.GinkgoTInterface, restConfig *rest.Config) error {
	testingContext, _ := NewTestingContext(restConfig)
	err := wait.Poll(time.Second*1, time.Minute*10, func() (done bool, err error) {
		rhmi, _ := GetRHMI(testingContext.Client, true)
		if rhmi.Status.Stage == "complete" {
			return true, nil
		}
		t.Logf("RHMI CR status.stage is: \"%s\". Waiting for: \"complete\"", rhmi.Status.Stage)
		time.Sleep(time.Second * 10)
		return false, nil

	})
	if err != nil {
		return fmt.Errorf("error waiting for RHMI CR status.stage to be \"complete\"")
	}
	return nil
}

func GetPlatformType(ctx *TestingContext) string {
	infra, err := pkgresources.GetClusterInfrastructure(goctx.TODO(), ctx.Client)
	if err != nil || infra.Status.PlatformStatus == nil {
		fmt.Println("can't retrieve cluster infrastructure")
		return ""
	}
	return string(infra.Status.PlatformStatus.Type)
}

func getRoutes(ctx *TestingContext, routeName string, namespace string) (routev1.Route, error) {
	routes := &routev1.RouteList{}

	routeFound := routev1.Route{}
	err := ctx.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{
		Namespace: namespace,
	})

	if err != nil {
		return routeFound, fmt.Errorf("failed to get 3scale routes with error: %v", err)
	}

	for _, route := range routes.Items {
		if strings.Contains(route.Spec.Host, routeName) {
			routeFound = route
		}
	}

	return routeFound, nil
}

func getToken(ctx *TestingContext, namespace, tokenType, objectMetaName string) (*string, error) {
	token := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
	}
	err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: token.Name, Namespace: namespace}, token)
	if err != nil {
		return nil, err
	}
	accessToken := string(token.Data[tokenType])
	return &accessToken, nil
}

func makeProject(ctx *TestingContext, namespace string) (*v1.Project, error) {
	project := &v1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := ctx.Client.Create(goctx.TODO(), project); err != nil {
		return project, fmt.Errorf("failed to create testing namespace with error: %v", err)
	}

	return project, nil
}

func genSecret(ctx *TestingContext, datamap map[string][]byte, secretName string, namespace string) (*corev1.Secret, error) {
	secretRef := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: datamap,
	}
	if err := ctx.Client.Create(goctx.TODO(), secretRef); err != nil {
		return secretRef, fmt.Errorf("failed to create secret with error: %v", err)
	}

	return secretRef, nil
}
