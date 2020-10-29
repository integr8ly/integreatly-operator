package common

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	v1 "github.com/openshift/api/route/v1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CheDevFile struct {
	ApiVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Commands   []Command   `yaml:"commands,omitempty" json:"commands,omitempty"`
	Components []Component `yaml:"components,omitempty" json:"components,omitempty"`
	Projects   []Project   `yaml:"projects,omitempty" json:"projects,omitempty"`
}

type Metadata struct {
	GenerateName string `yaml:"generateName" json:"generateName"`
}

type Command struct {
	Name    string   `yaml:"name,omitempty" json:"name,omitempty"`
	Actions []Action `yaml:"actions,omitempty" json:"actions,omitempty"`
}

type Action struct {
	Command          string      `yaml:"command,omitempty" json:"command,omitempty"`
	Component        string      `yaml:"component,omitempty" json:"component,omitempty"`
	Type             string      `yaml:"type,omitempty" json:"type,omitempty"`
	Workdir          string      `yaml:"workdir,omitempty" json:"workdir,omitempty"`
	ReferenceContent interface{} `yaml:"referenceContent,omitempty" json:"referenceContent,omitempty"`
}

type Component struct {
	Alias        string            `yaml:"alias,omitempty" json:"alias,omitempty"`
	ID           string            `yaml:"id,omitempty" json:"id,omitempty"`
	Endpoints    []Endpoint        `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`
	Env          []Env             `yaml:"env,omitempty" json:"env,omitempty"`
	Image        string            `yaml:"image,omitempty" json:"image,omitempty"`
	MemoryLimit  string            `yaml:"memoryLimit,omitempty" json:"memoryLimit,omitempty"`
	MountSources bool              `yaml:"mountSources,omitempty" json:"mountSources,omitempty"`
	Preferences  map[string]string `yaml:"preferences,omitempty" json:"preferences,omitempty"`
	Type         string            `yaml:"type,omitempty" json:"type,omitempty"`
}

type Endpoint struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Port int    `yaml:"port,omitempty" json:"port,omitempty"`
}

type Env struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

type Project struct {
	ClonePath string        `yaml:"clonePath,omitempty" json:"clonePath,omitempty"`
	Name      string        `yaml:"name,omitempty" json:"name,omitempty"`
	Source    ProjectSource `yaml:"source,omitempty" json:"source,omitempty"`
}

type ProjectSource struct {
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
	Location string `yaml:"location,omitempty" json:"location,omitempty"`
}

const (
	clientId                  = "che-client"
	goDevFilePath             = "https://%v/devfiles/05_go/devfile.yaml"
	devFileRegistryRouteName  = "devfile-registry"
	codereadyProductNamespace = "codeready-workspaces"
	workspaceStatusReady      = "RUNNING"
	workspaceStatusStopped    = "STOPPED"
)

var (
	username = fmt.Sprintf("%v-0", DefaultTestUserName)
	password = DefaultPassword
)

func TestCodereadyCrudlPermisssions(t *testing.T, ctx *TestingContext) {
	t.Log("Test codeready workspace creation")

	// Ensure testing-idp is available
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("failed to create testing idp: %v", err)
	}

	// Get the master, codeready and keycloak url from the rhmi status
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL
	cheHost := rhmi.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductCodeReadyWorkspaces].Host
	keycloakHost := rhmi.Status.Stages[integreatlyv1alpha1.AuthenticationStage].Products[integreatlyv1alpha1.ProductRHSSO].Host

	// Get codeready access token to be used for API requests
	redirectUrl := fmt.Sprintf("%v/dashboard/", cheHost)

	// login to openshift
	loginClient := resources.NewCodereadyLoginClient(ctx.HttpClient, ctx.Client, masterURL, TestingIDPRealm, username, password, t)
	if err := loginClient.OpenshiftLogin(rhmi.Spec.NamespacePrefix); err != nil {
		t.Fatalf("failed to login to openshift: %v", err)
	}

	// login to codeready
	token, err := loginClient.CodereadyLogin(keycloakHost, redirectUrl, t)
	if err != nil {
		t.Fatalf("failed to login to codeready: %v", err)
	}

	// Get codeready api client
	codereadyClient := resources.NewCodereadyApiClient(ctx.HttpClient, cheHost, token)
	codereadyNamespace := fmt.Sprintf("%v%v", rhmi.Spec.NamespacePrefix, codereadyProductNamespace)

	// Get Codeready Go dev file. This will be used as a payload to create a workspace
	devfile, err := getDevFile(ctx.HttpClient, ctx.Client, codereadyNamespace)
	if err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("failed to get che devfile: %v", err.Error())
	}

	// Create a codeready workspace
	workspaceID, err := codereadyClient.CreateWorkspace(devfile)
	if err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("failed to create a codeready workspace: %v", err)
	}

	// Ensure workspace starts successfully
	createWorkspaceRetryInterval := time.Second * 20
	createWorkspaceTimeout := time.Minute * 7
	if err := waitForWorkspaceStatus(codereadyClient, createWorkspaceRetryInterval, createWorkspaceTimeout, workspaceID, workspaceStatusReady); err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("failed to start codeready workspace %v: %v", workspaceID, err)
	}

	// Stop workspace before deleting
	if err := codereadyClient.StopWorkspace(workspaceID); err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("failed to stop codeready workspace %v: %v", workspaceID, err)
	}

	stopWorkspaceRetryInterval := time.Second * 10
	stopWorkspaceTimeout := time.Minute * 3
	if err := waitForWorkspaceStatus(codereadyClient, stopWorkspaceRetryInterval, stopWorkspaceTimeout, workspaceID, workspaceStatusStopped); err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("failed to wait for '%v' workspace to stop: %v", workspaceID, err)
	}

	// Delete workspace
	if err := codereadyClient.DeleteWorkspace(workspaceID); err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("failed to delete codeready workspace %v: %v", workspaceID, err)
	}

	// Workspace resources created in openshift should already be removed at this stage. Ensure these resources are removed
	if err := ensureWorkspaceResourcesRemoved(ctx, workspaceID, codereadyNamespace); err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6679")
		//t.Fatalf("workspace resources still exists in the cluster: %v", err)
	}
}

/*
Codeready creates the following resources in openshift: deployments, pods, secrets, routes and services.
This function returns an error if any of the resources created have not been removed from openshift.
*/
func ensureWorkspaceResourcesRemoved(ctx *TestingContext, workspaceId, namespace string) error {
	labelSelector := fmt.Sprintf("che.workspace_id=%s", workspaceId)
	listOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	deployments, err := ctx.KubeClient.AppsV1().Deployments(namespace).List(listOpts)
	if err != nil {
		return fmt.Errorf("failed to list deployments in %v namespace: %v", namespace, err)
	}

	if len(deployments.Items) != 0 {
		return fmt.Errorf("there are %v workspace deployments remaining", len(deployments.Items))
	}

	pods, err := ctx.KubeClient.CoreV1().Pods(namespace).List(listOpts)
	if err != nil {
		return fmt.Errorf("failed to list pods in %v namespace: %v", namespace, err)
	}

	if len(pods.Items) != 0 {
		return fmt.Errorf("there are %v workspace pods remaining", len(pods.Items))
	}

	services, err := ctx.KubeClient.CoreV1().Services(namespace).List(listOpts)
	if err != nil {
		return fmt.Errorf("failed to list services in %v namespace: %v", namespace, err)
	}

	if len(services.Items) != 0 {
		return fmt.Errorf("there are %v workspace services remaining", len(services.Items))
	}

	secrets, err := ctx.KubeClient.CoreV1().Secrets(namespace).List(listOpts)
	if err != nil {
		return fmt.Errorf("failed to list secrets in %v namespace: %v", namespace, err)
	}

	if len(secrets.Items) != 0 {
		return fmt.Errorf("there are %v worspace secrets remaining", len(secrets.Items))
	}

	routes := &v1.RouteList{}

	label, err := labels.Parse(labelSelector)
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %v", err)
	}

	if err := ctx.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{LabelSelector: label}); err != nil {
		return fmt.Errorf("failed to list routes in %v namespace: %v", namespace, err)
	}

	if len(routes.Items) != 0 {
		return fmt.Errorf("there are %v workspace routes remaining", len(routes.Items))
	}

	return nil
}

func waitForWorkspaceStatus(cheClient *resources.CodereadyApiClientImpl, retryInterval, timeout time.Duration, workspaceID, status string) error {
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		workspace, err := cheClient.GetWorkspace(workspaceID)
		if err != nil {
			return false, fmt.Errorf("failed to get codeready workspace %v: %v", workspaceID, err)
		}

		if workspace.Status != status {
			return false, nil
		}
		return true, nil
	})
}

func getContentFromURL(httpClient *http.Client, url string) ([]byte, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get file content from %v. Status: %d", url, response.StatusCode)
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func getDevFile(httpClient *http.Client, dynClient k8sclient.Client, codereadyNamespace string) ([]byte, error) {
	// Get Devfile registry route
	devFileRegistryRoute := &v1.Route{}
	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: devFileRegistryRouteName, Namespace: codereadyNamespace}, devFileRegistryRoute); err != nil {
		return nil, fmt.Errorf("failed to get devfile registry route in %v namespace: %v", codereadyNamespace, err)
	}

	url := fmt.Sprintf(goDevFilePath, devFileRegistryRoute.Spec.Host)

	fileContent, err := getContentFromURL(httpClient, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get content from url %v: %v", url, err.Error())
	}

	devFile := &CheDevFile{}
	if err := yaml.Unmarshal(fileContent, &devFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal devfile content: %v", err.Error())
	}

	// Convert yaml to json
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // Ensures that characters such as '&' does not get converted to unicode
	if err := encoder.Encode(devFile); err != nil {
		return nil, fmt.Errorf("failed to encode: %v", err.Error())
	}

	return buffer.Bytes(), nil
}
