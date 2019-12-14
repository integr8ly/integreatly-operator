package e2e

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	operator "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	retryInterval                    = time.Second * 5
	timeout                          = time.Second * 75
	deploymentRetryInterval          = time.Second * 30
	deploymentTimeout                = time.Minute * 20
	cleanupRetryInterval             = time.Second * 1
	cleanupTimeout                   = time.Second * 5
	installationCleanupRetryInterval = time.Second * 20
	installationCleanupTimeout       = time.Minute * 8 //Longer timeout required to allow for finalizers to execute
	intlyNamespacePrefix             = "rhmi-"
	namespaceLabel                   = "integreatly"
	installationName                 = "integreatly-operator"
	bootstrapStage                   = "bootstrap"
	monitoringStage                  = "monitoring"
	authenticationStage              = "authentication"
	productsStage                    = "products"
	solutionExplorerStage            = "solution-explorer"
)

func TestIntegreatly(t *testing.T) {
	installationList := &operator.InstallationList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, installationList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("integreatly", func(t *testing.T) {
		t.Run("Cluster", IntegreatlyCluster)
	})

}

func waitForProductDeployment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, product, deploymentName string) error {
	namespace := intlyNamespacePrefix + product
	t.Logf("Checking %s:%s", namespace, deploymentName)

	start := time.Now()
	err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, 1, deploymentRetryInterval, deploymentTimeout)
	if err != nil {
		return err
	}

	end := time.Now()
	elapsed := end.Sub(start)

	t.Logf("%s:%s up, waited %d", namespace, deploymentName, elapsed)
	return nil
}

func integreatlyMonitoringTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	// Define the json output of the prometheus api call
	type Labels struct {
		Alertname string `json:"alertname,omitempty"`
		Severity  string `json:"severity,omitempty"`
	}

	type Annotations struct {
		Message string `json:"message,omitempty"`
	}

	type Alerts struct {
		Labels      Labels      `json:"labels,omitempty"`
		State       string      `json:"state,omitempty"`
		Annotations Annotations `json:"annotations,omitempty"`
		ActiveAt    string      `json:"activeAt,omitempty"`
		Value       string      `json:"value,omitempty"`
	}

	type Data struct {
		Alerts []Alerts `json:"alerts,omitempty"`
	}

	type Output struct {
		Status string `json:"status"`
		Data   Data   `json:"data"`
	}

	output, err := execToPod("curl localhost:9090/api/v1/alerts",
		"prometheus-application-monitoring-0",
		intlyNamespacePrefix+"middleware-monitoring",
		"prometheus", f)
	if err != nil {
		return fmt.Errorf("failed to exec to pod: %s", err)
	}

	var promApiCallOutput Output
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}

	// Check if any alerts other than DeadMansSwitch are firing or pending
	var firingalerts []string
	var pendingalerts []string
	var deadmanswitchfiring = false
	for a := 0; a < len(promApiCallOutput.Data.Alerts); a++ {
		if promApiCallOutput.Data.Alerts[a].Labels.Alertname == "DeadMansSwitch" && promApiCallOutput.Data.Alerts[a].State == "firing" {
			deadmanswitchfiring = true
		}
		if promApiCallOutput.Data.Alerts[a].Labels.Alertname != "DeadMansSwitch" {
			// ESPodCount will always fail since that alert is for OSD
			// so it shouldn't cause our test to fail
			if promApiCallOutput.Data.Alerts[a].Labels.Alertname == "KubePodCrashLooping" {
				continue
			}
			if promApiCallOutput.Data.Alerts[a].State == "firing" {
				firingalerts = append(firingalerts, promApiCallOutput.Data.Alerts[a].Labels.Alertname)
			}
			if promApiCallOutput.Data.Alerts[a].State == "pending" {
				pendingalerts = append(pendingalerts, promApiCallOutput.Data.Alerts[a].Labels.Alertname)
			}
		}
	}

	var status []string
	if len(firingalerts) > 0 {
		falert := fmt.Sprint(string(len(firingalerts))+"Firing alerts: ", firingalerts)
		status = append(status, falert)
	}
	if len(pendingalerts) > 0 {
		palert := fmt.Sprint(string(len(pendingalerts))+"Pending alerts: ", pendingalerts)
		status = append(status, palert)
	}
	if deadmanswitchfiring == false {
		dms := fmt.Sprint("DeadMansSwitch is not firing")
		status = append(status, dms)
	}

	if len(status) > 0 {
		return fmt.Errorf("alert tests failed: %s", status)
	}

	t.Logf("No unexpected alerts found")
	return nil
}

func execToPod(command string, podname string, namespace string, container string, f *framework.Framework) (string, error) {
	req := f.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podname).
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

	exec, err := remotecommand.NewSPDYExecutor(f.KubeConfig, "POST", req.URL())
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

func getConfigMap(name string, namespace string, f *framework.Framework) (map[string]string, error) {
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	key := client.ObjectKey{
		Name:      configmap.GetName(),
		Namespace: configmap.GetNamespace(),
	}
	err := f.Client.Get(goctx.TODO(), key, configmap)
	if err != nil {
		return map[string]string{}, fmt.Errorf("could not get configmap: %configmapname", err)
	}

	return configmap.Data, nil
}

func integreatlyManagedTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %deploymentName", err)
	}

	// wait for bootstrap phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, bootstrapStage)
	if err != nil {
		return err
	}

	// wait for middleware-monitoring to deploy
	err = waitForProductDeployment(t, f, ctx, "middleware-monitoring", "application-monitoring-operator")
	if err != nil {
		return err
	}

	// wait for authentication phase to complete (15 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, monitoringStage)
	if err != nil {
		return err
	}

	// wait for keycloak-operator to deploy
	err = waitForProductDeployment(t, f, ctx, "rhsso", "keycloak-operator")
	if err != nil {
		return err
	}

	// wait for authentication phase to complete (15 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, authenticationStage)
	if err != nil {
		return err
	}

	//Product Stage - verify operators deploy
	products := map[string]string{
		"3scale":               "3scale-operator",
		"amq-online":           "enmasse-operator",
		"codeready-workspaces": "codeready-operator",
		"fuse":                 "syndesis-operator",
		"user-sso":             "keycloak-operator",
		"ups":                  "unifiedpush-operator",
	}
	for product, deploymentName := range products {
		err = waitForProductDeployment(t, f, ctx, product, deploymentName)
		if err != nil {
			return err
		}
	}

	// wait for products phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout*2, productsStage)
	if err != nil {
		return err
	}

	// wait for solution-explorer operator to deploy
	err = waitForProductDeployment(t, f, ctx, "solution-explorer", "tutorial-web-app-operator")
	if err != nil {
		return err
	}

	// wait for solution-explorer phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, solutionExplorerStage)
	if err != nil {
		return err
	}

	// check namespaces labelled correctly
	expectedNamespaces := []string{
		"3scale",
		"amq-online",
		"codeready-workspaces",
		"fuse",
		"middleware-monitoring",
		"rhsso",
		"solution-explorer",
		"ups",
		"user-sso",
	}
	err = checkIntegreatlyNamespaceLabels(t, f, expectedNamespaces, namespaceLabel)
	if err != nil {
		return err
	}

	// check auth stage operator versions
	stage := operator.StageName("authentication")
	authOperators := map[string]string{
		"rhsso": string(operator.OperatorVersionRHSSO),
	}
	err = checkOperatorVersions(t, f, namespace, stage, authOperators)
	if err != nil {
		return err
	}

	// check cloud resources stage operator versions
	stage = operator.StageName("cloud-resources")
	resouceOperators := map[string]string{
		"cloud-resources": string(operator.OperatorVersionCloudResources),
	}
	err = checkOperatorVersions(t, f, namespace, stage, resouceOperators)
	if err != nil {
		return err
	}

	// check monitoring stage operator versions
	stage = operator.StageName("monitoring")
	monitoringOperators := map[string]string{
		"monitoring": string(operator.OperatorVersionMonitoring),
	}
	err = checkOperatorVersions(t, f, namespace, stage, monitoringOperators)
	if err != nil {
		return err
	}

	// check products stage operator versions
	stage = operator.StageName("products")
	productOperators := map[string]string{
		"3scale":               string(operator.OperatorVersion3Scale),
		"amqonline":            string(operator.OperatorVersionAMQOnline),
		"codeready-workspaces": string(operator.OperatorVersionCodeReadyWorkspaces),
		"fuse-on-openshift":    string(operator.OperatorVersionFuse),
		"ups":                  string(operator.OperatorVersionUPS),
		"rhssouser":            string(operator.OperatorVersionRHSSOUser),
	}
	err = checkOperatorVersions(t, f, namespace, stage, productOperators)
	if err != nil {
		return err
	}

	// check auth stage operand versions
	stage = operator.StageName("authentication")
	authOperands := map[string]string{
		"rhsso": "v7.3.2.GA",
	}
	err = checkOperandVersions(t, f, namespace, stage, authOperands)
	if err != nil {
		return err
	}

	// check cloud resources stage operand versions
	stage = operator.StageName("cloud-resources")
	resouceOperands := map[string]string{
		"cloud-resources": string(operator.VersionCloudResources),
	}
	err = checkOperandVersions(t, f, namespace, stage, resouceOperands)
	if err != nil {
		return err
	}

	// check monitoring stage operand versions
	stage = operator.StageName("monitoring")
	monitoringOperands := map[string]string{
		"monitoring": string(operator.VersionMonitoring),
	}
	err = checkOperandVersions(t, f, namespace, stage, monitoringOperands)
	if err != nil {
		return err
	}

	// check products stage operands versions
	stage = operator.StageName("products")
	productOperands := map[string]string{
		"3scale":               string(operator.Version3Scale),
		"amqonline":            string(operator.VersionAMQOnline),
		"codeready-workspaces": string(operator.VersionCodeReadyWorkspaces),
		"fuse-on-openshift":    string(operator.VersionFuseOnOpenshift),
		"ups":                  string(operator.VersionUps),
		"rhssouser":            "v7.3.2.GA",
	}
	err = checkOperandVersions(t, f, namespace, stage, productOperands)
	if err != nil {
		return err
	}

	// check routes were created by checking hardcoded number of routes
	// would be nice if expected routes can be dynamically discovered
	expectedRoutes := map[string]int{
		"3scale":                6,
		"amq-online":            2,
		"codeready-workspaces":  1,
		"fuse":                  2,
		"middleware-monitoring": 3,
		"rhsso":                 1,
		"solution-explorer":     1,
		"ups":                   1,
		"user-sso":              1,
	}

	for product, numberRoutes := range expectedRoutes {
		err = checkRoutes(t, f, product, numberRoutes)
		if err != nil {
			return err
		}
	}

	// check no failed PVCs
	pvcNamespaces := []string{
		"3scale",
		"fuse",
		"rhsso",
		"solution-explorer",
		"ups",
		"user-sso",
	}
	err = checkPvcs(t, f, namespace, pvcNamespaces)
	if err != nil {
		return err
	}

	return err
}

func checkIntegreatlyNamespaceLabels(t *testing.T, f *framework.Framework, namespaces []string, label string) error {
	for _, namespaceName := range namespaces {
		namespace := &corev1.Namespace{}
		err := f.Client.Get(goctx.TODO(), client.ObjectKey{Name: intlyNamespacePrefix + namespaceName}, namespace)
		if err != nil {
			return fmt.Errorf("Error getting namespace: %v from cluster: %w", namespaceName, err)
		}
		value, ok := namespace.Labels[label]
		if !ok || value != "true" {
			return fmt.Errorf("Incorrect %v label on integreatly namespace: %v. Expected: true. Got: %v", label, namespaceName, value)
		}
	}
	return nil
}

func checkOperatorVersions(t *testing.T, f *framework.Framework, namespace string, stage operator.StageName, operatorVersions map[string]string) error {
	installation := &operator.Installation{}

	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: namespace}, installation)
	if err != nil {
		return fmt.Errorf("Error getting installation CR from cluster when checking operator versions: %w", err)
	}

	for product, version := range operatorVersions {
		clusterVersion := installation.Status.Stages[stage].Products[operator.ProductName(product)].OperatorVersion
		if clusterVersion != operator.OperatorVersion(version) {
			return fmt.Errorf("Error with version of %s operator deployed on cluster. Expected %s. Got %s", product, version, clusterVersion)
		}
	}

	return nil
}

func checkOperandVersions(t *testing.T, f *framework.Framework, namespace string, stage operator.StageName, operandVersions map[string]string) error {
	installation := &operator.Installation{}

	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: namespace}, installation)
	if err != nil {
		return fmt.Errorf("Error getting installation CR from cluster when checking operand versions: %w", err)
	}

	for product, version := range operandVersions {
		clusterVersion := installation.Status.Stages[stage].Products[operator.ProductName(product)].Version
		if clusterVersion != operator.ProductVersion(version) {
			return fmt.Errorf("Error with version of %s deployed on cluster. Expected %s. Got %s", product, version, clusterVersion)
		}
	}

	return nil
}

func checkRoutes(t *testing.T, f *framework.Framework, product string, numberRoutes int) error {
	routes := &routev1.RouteList{}
	err := f.Client.List(goctx.TODO(), routes, &client.ListOptions{Namespace: intlyNamespacePrefix + product})
	if err != nil {
		return fmt.Errorf("Error getting routes for %s namespace: %w", product, err)
	}
	if len(routes.Items) != numberRoutes {
		return fmt.Errorf("Expected %v in %v%v namespace. Found %v", numberRoutes, intlyNamespacePrefix, product, len(routes.Items))
	}
	return nil
}

func checkPvcs(t *testing.T, f *framework.Framework, s string, pvcNamespaces []string) error {
	for _, pvcNamespace := range pvcNamespaces {
		pvcs := &corev1.PersistentVolumeClaimList{}
		err := f.Client.List(goctx.TODO(), pvcs, &client.ListOptions{Namespace: intlyNamespacePrefix + pvcNamespace})
		if err != nil {
			return fmt.Errorf("Error getting PVCs for namespace: %v. %w", pvcNamespace, err)
		}
		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase != "Bound" {
				return fmt.Errorf("Error with pvc: %v. Status: %v", pvc.Name, pvc.Status.Phase)
			}
		}
	}
	return nil
}

func waitForInstallationStageCompletion(t *testing.T, f *framework.Framework, namespace string, retryInterval, timeout time.Duration, phase string) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		installation := &operator.Installation{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: namespace}, installation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s installation\n", installationName)
				return false, nil
			}
			return false, err
		}

		phaseStatus := fmt.Sprintf("%#v", installation.Status.Stages[operator.StageName(phase)])
		if strings.Contains(phaseStatus, "completed") {
			return true, nil
		}

		t.Logf("Waiting for completion of %s\n", phase)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("%s phase completed \n", phase)
	return nil
}

func IntegreatlyCluster(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for integreatly-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "integreatly-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
	// check that all of the operators deploy and all of the installation phases complete
	if err = integreatlyManagedTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}

	t.Log("Waiting for alerts to normalise")
	time.Sleep(5 * time.Minute)

	if err = integreatlyMonitoringTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
