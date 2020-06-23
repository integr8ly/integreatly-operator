package common

import (
	"bytes"
	goctx "context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/resources"

	"golang.org/x/net/publicsuffix"
	"gopkg.in/yaml.v2"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	cached "k8s.io/client-go/discovery/cached"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func execToPod(command string, podName string, namespace string, container string, ctx *TestingContext) (string, error) {
	req := ctx.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(ctx.KubeConfig, "POST", req.URL())
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
		return "", fmt.Errorf("error in Stream: %v", err)
	}
	return stdout.String(), nil
}

// difference one-way diff that return strings in sliceSource that are not in sliceTarget
func difference(sliceSource, sliceTarget []string) []string {
	// create an empty lookup map with keys from sliceTarget
	diffSourceLookupMap := make(map[string]struct{}, len(sliceTarget))
	for _, item := range sliceTarget {
		diffSourceLookupMap[item] = struct{}{}
	}
	// use the lookup map to find items in sliceSource that are not in sliceTarget
	// and store them in a diff slice
	var diff []string
	for _, item := range sliceSource {
		if _, found := diffSourceLookupMap[item]; !found {
			diff = append(diff, item)
		}
	}
	return diff
}

// Is the cluster using on cluster or external storage
func isClusterStorage(ctx *TestingContext) (bool, error) {
	rhmi := &integreatlyv1alpha1.RHMI{}
	// get the RHMI custom resource to check what storage type is being used
	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: InstallationName, Namespace: RHMIOperatorNamespace}, rhmi)
	if err != nil {
		return true, fmt.Errorf("error getting RHMI CR: %v", err)
	}

	if rhmi.Spec.UseClusterStorage == "true" {
		return true, nil
	}
	return false, nil
}

// returns rhmi
func getRHMI(client dynclient.Client) (*integreatlyv1alpha1.RHMI, error) {
	rhmi := &integreatlyv1alpha1.RHMI{}
	if err := client.Get(goctx.TODO(), types.NamespacedName{Name: InstallationName, Namespace: RHMIOperatorNamespace}, rhmi); err != nil {
		return nil, fmt.Errorf("error getting RHMI CR: %w", err)
	}
	return rhmi, nil
}

func NewTestingContext(kubeConfig *rest.Config) (*TestingContext, error) {
	kubeclient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the kubeclient: %v", err)
	}

	apiextensions, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build the apiextension client: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add cgo scheme to runtime scheme: (%v)", err)
	}
	if err := extscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add api extensions scheme to runtime scheme: (%v)", err)
	}
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add integreatly scheme to runtime scheme: (%v)", err)
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()

	dynClient, err := dynclient.New(kubeConfig, dynclient.Options{Scheme: scheme, Mapper: restMapper})
	if err != nil {
		return nil, fmt.Errorf("failed to build the dynamic client: %v", err)
	}

	selfSignedCerts, err := HasSelfSignedCerts(kubeConfig.Host, http.DefaultClient)
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

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: selfSignedCerts},
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

func writeObjToYAMLFile(obj interface{}, out string) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(out, data, 0644)
}

func WriteRHMICRToFile(client dynclient.Client, file string) error {
	if rhmi, err := getRHMI(client); err != nil {
		return err
	} else {
		return writeObjToYAMLFile(rhmi, file)
	}
}

// Common function to perform CRUDL and verifying their expected permissions
func verifyCRUDLPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient, expectedPermission ExpectedPermissions) {
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
	defer resp.Body.Close()
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
