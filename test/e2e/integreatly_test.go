package e2e

import (
	"bytes"
	"context"
	goctx "context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/test/common"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

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
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	retryInterval                = time.Second * 5
	timeout                      = time.Second * 75
	deploymentRetryInterval      = time.Second * 30
	deploymentTimeout            = time.Minute * 20
	cleanupRetryInterval         = time.Second * 1
	cleanupTimeout               = time.Second * 5
	intlyNamespacePrefix         = "redhat-rhmi-"
	namespaceLabel               = "integreatly"
	bootStrapStageTimeout        = time.Minute * 5
	cloudResourcesStageTimeout   = time.Minute * 10
	monitoringStageTimeout       = time.Minute * 10
	authenticationStageTimeout   = time.Minute * 30
	productsStageTimout          = time.Minute * 30
	solutionExplorerStageTimeout = time.Minute * 10
)

func TestIntegreatly(t *testing.T) {
	err := framework.AddToFrameworkScheme(apis.AddToScheme, &integreatlyv1alpha1.RHMIList{})

	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global

	apiextensions, err := clientset.NewForConfig(f.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	testingContext := &common.TestingContext{
		Client:          f.Client.Client,
		KubeConfig:      f.KubeConfig,
		KubeClient:      f.KubeClient,
		ExtensionClient: apiextensions,
	}

	// run subtests
	t.Run("integreatly", func(t *testing.T) {
		for _, test := range common.ALL_TESTS {
			t.Run(test.Description(), func(t *testing.T) {
				test.Test(t, testingContext)
			})
		}

		t.Run("Cluster", func(t *testing.T) {
			IntegreatlyCluster(t, f, ctx)
		})

		for _, test := range common.AFTER_INSTALL_TESTS {
			t.Run(test.Description(), func(t *testing.T) {
				test.Test(t, testingContext)
			})
		}
	})

}

func waitForProductDeployment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, product, deploymentName string) error {
	namespace := ""
	if deploymentName != "enmasse-operator" {
		namespace = intlyNamespacePrefix + product + "-operator"
	} else {
		namespace = intlyNamespacePrefix + product
	}
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
	type apiResponse struct {
		Status    string                 `json:"status"`
		Data      json.RawMessage        `json:"data"`
		ErrorType prometheusv1.ErrorType `json:"errorType"`
		Error     string                 `json:"error"`
		Warnings  []string               `json:"warnings,omitempty"`
	}

	// Get active alerts
	output, err := execToPod("curl localhost:9090/api/v1/alerts",
		"prometheus-application-monitoring-0",
		intlyNamespacePrefix+"middleware-monitoring-operator",
		"prometheus", f)
	if err != nil {
		return fmt.Errorf("failed to exec to pod: %s", err)
	}

	var promApiCallOutput apiResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}
	var alertsResult prometheusv1.AlertsResult
	err = json.Unmarshal(promApiCallOutput.Data, &alertsResult)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}

	// Check if any alerts other than DeadMansSwitch are firing or pending
	var firingalerts []string
	var pendingalerts []string
	var deadmanswitchfiring = false
	for _, alert := range alertsResult.Alerts {
		if alert.Labels["alertname"] == "DeadMansSwitch" && alert.State == "firing" {
			deadmanswitchfiring = true
		}
		if alert.Labels["alertname"] != "DeadMansSwitch" {
			if alert.State == "firing" {
				firingalerts = append(firingalerts, string(alert.Labels["alertname"]))
			}
			if alert.State == "pending" {
				pendingalerts = append(pendingalerts, string(alert.Labels["alertname"]))
			}
		}
	}

	// Get all rules
	output, err = execToPod("curl localhost:9090/api/v1/rules",
		"prometheus-application-monitoring-0",
		intlyNamespacePrefix+"middleware-monitoring-operator",
		"prometheus", f)
	if err != nil {
		return fmt.Errorf("failed to exec to pod: %s", err)
	}

	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}
	var rulesResult prometheusv1.RulesResult
	err = json.Unmarshal([]byte(promApiCallOutput.Data), &rulesResult)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}

	// Check that at least one integreatly alert is present
	var intlyalertpresent = false
	for _, group := range rulesResult.Groups {
		for _, rule := range group.Rules {
			switch v := rule.(type) {
			case prometheusv1.RecordingRule:
				fmt.Print("got a recording rule")
			case prometheusv1.AlertingRule:
				if rule.(prometheusv1.AlertingRule).Name == "KubePodCrashLooping" {
					intlyalertpresent = true
				}
			default:
				fmt.Printf("unknown rule type %s", v)
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
	if intlyalertpresent == false {
		ialert := fmt.Sprint("KubePodCrashLooping is not present")
		status = append(status, ialert)
	}

	if len(status) > 0 {
		return fmt.Errorf("alert tests failed: %s", status)
	}

	t.Logf("No unexpected alerts found")
	return nil
}

func integreatlyGrafanaTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	pods := &corev1.PodList{}
	opts := []k8sclient.ListOption{
		k8sclient.InNamespace(intlyNamespacePrefix + "middleware-monitoring-operator"),
		k8sclient.MatchingLabels{"app": "grafana"},
	}
	err := f.Client.List(goctx.TODO(), pods, opts...)
	if err != nil {
		return fmt.Errorf("failed to list pods: %s", err)
	}
	if len(pods.Items) != 1 {
		return fmt.Errorf("grafana pod not found")
	}

	type Dashboard struct {
		Title string `json:"title"`
	}

	type Output []Dashboard

	output, err := execToPod("curl localhost:3000/api/search",
		pods.Items[0].ObjectMeta.Name,
		intlyNamespacePrefix+"middleware-monitoring-operator",
		"grafana", f)
	if err != nil {
		return fmt.Errorf("failed to exec to pod: %s", err)
	}

	var apiCallOutput Output
	err = json.Unmarshal([]byte(output), &apiCallOutput)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}

	if len(apiCallOutput) == 0 {
		return fmt.Errorf("grafana dashboard not found : %v", apiCallOutput)
	}
	t.Logf("Grafana dashboard found: %v", apiCallOutput)
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
	key := k8sclient.ObjectKey{
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

	// wait for cloud resource phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, cloudResourcesStageTimeout, string(integreatlyv1alpha1.CloudResourcesStage))
	if err != nil {
		return err
	}

	// wait for cloud resource to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductCloudResources), "cloud-resource-operator")
	if err != nil {
		return err
	}

	// wait for bootstrap phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, bootStrapStageTimeout, string(integreatlyv1alpha1.BootstrapStage))
	if err != nil {
		return err
	}

	// wait for middleware-monitoring to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductMonitoring), "application-monitoring-operator")
	if err != nil {
		return err
	}

	// wait for middleware-monitoring phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, monitoringStageTimeout, string(integreatlyv1alpha1.MonitoringStage))
	if err != nil {
		return err
	}

	// wait for keycloak-operator to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductRHSSO), "keycloak-operator")
	if err != nil {
		return err
	}

	// wait for authentication phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, authenticationStageTimeout, string(integreatlyv1alpha1.AuthenticationStage))
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
		"apicurito":            "apicurito-operator",
	}
	for product, deploymentName := range products {
		err = waitForProductDeployment(t, f, ctx, product, deploymentName)
		if err != nil {
			return err
		}
	}

	// wait for products phase to complete (30 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, productsStageTimout, string(integreatlyv1alpha1.ProductsStage))
	if err != nil {
		return err
	}

	// wait for solution-explorer operator to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductSolutionExplorer), "tutorial-web-app-operator")
	if err != nil {
		return err
	}

	// wait for solution-explorer phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, solutionExplorerStageTimeout, string(integreatlyv1alpha1.SolutionExplorerStage))
	if err != nil {
		return err
	}

	// check namespaces labelled correctly
	expectedNamespaces := []string{
		"3scale",
		"3scale-operator",
		"amq-online",
		"apicurito",
		"apicurito-operator",
		"codeready-workspaces",
		"codeready-workspaces-operator",
		"fuse",
		"fuse-operator",
		"middleware-monitoring-operator",
		"rhsso",
		"rhsso-operator",
		"solution-explorer",
		"solution-explorer-operator",
		"ups",
		"ups-operator",
		"user-sso",
		"user-sso-operator",
	}
	err = checkIntegreatlyNamespaceLabels(t, f, expectedNamespaces, namespaceLabel)
	if err != nil {
		return err
	}

	// check auth stage operator versions
	authOperators := map[string]string{
		string(integreatlyv1alpha1.ProductRHSSO): string(integreatlyv1alpha1.OperatorVersionRHSSO),
	}
	err = checkOperatorVersions(t, f, namespace, integreatlyv1alpha1.AuthenticationStage, authOperators)
	if err != nil {
		return err
	}

	// check cloud resources stage operator versions
	resouceOperators := map[string]string{
		string(integreatlyv1alpha1.ProductCloudResources): string(integreatlyv1alpha1.OperatorVersionCloudResources),
	}
	err = checkOperatorVersions(t, f, namespace, integreatlyv1alpha1.CloudResourcesStage, resouceOperators)
	if err != nil {
		return err
	}

	// check monitoring stage operator versions
	monitoringOperators := map[string]string{
		string(integreatlyv1alpha1.ProductMonitoring): string(integreatlyv1alpha1.OperatorVersionMonitoring),
	}
	err = checkOperatorVersions(t, f, namespace, integreatlyv1alpha1.MonitoringStage, monitoringOperators)
	if err != nil {
		return err
	}

	// check products stage operator versions
	productOperators := map[string]string{
		string(integreatlyv1alpha1.Product3Scale):              string(integreatlyv1alpha1.OperatorVersion3Scale),
		string(integreatlyv1alpha1.ProductAMQOnline):           string(integreatlyv1alpha1.OperatorVersionAMQOnline),
		string(integreatlyv1alpha1.ProductApicurito):           string(integreatlyv1alpha1.OperatorVersionApicurito),
		string(integreatlyv1alpha1.ProductCodeReadyWorkspaces): string(integreatlyv1alpha1.OperatorVersionCodeReadyWorkspaces),
		string(integreatlyv1alpha1.ProductFuseOnOpenshift):     string(integreatlyv1alpha1.OperatorVersionFuse),
		string(integreatlyv1alpha1.ProductUps):                 string(integreatlyv1alpha1.OperatorVersionUPS),
		string(integreatlyv1alpha1.ProductRHSSOUser):           string(integreatlyv1alpha1.OperatorVersionRHSSOUser),
	}
	err = checkOperatorVersions(t, f, namespace, integreatlyv1alpha1.ProductsStage, productOperators)
	if err != nil {
		return err
	}

	// check authentication stage operand versions
	authOperands := map[string]string{
		string(integreatlyv1alpha1.ProductRHSSO): string(integreatlyv1alpha1.VersionRHSSO),
	}
	err = checkOperandVersions(t, f, namespace, integreatlyv1alpha1.AuthenticationStage, authOperands)
	if err != nil {
		return err
	}

	// check cloud resources stage operand versions
	resouceOperands := map[string]string{
		string(integreatlyv1alpha1.ProductCloudResources): string(integreatlyv1alpha1.VersionCloudResources),
	}
	err = checkOperandVersions(t, f, namespace, integreatlyv1alpha1.CloudResourcesStage, resouceOperands)
	if err != nil {
		return err
	}

	// check monitoring stage operand versions
	monitoringOperands := map[string]string{
		string(integreatlyv1alpha1.ProductMonitoring): string(integreatlyv1alpha1.VersionMonitoring),
	}
	err = checkOperandVersions(t, f, namespace, integreatlyv1alpha1.MonitoringStage, monitoringOperands)
	if err != nil {
		return err
	}

	// check products stage operands versions
	productOperands := map[string]string{
		string(integreatlyv1alpha1.Product3Scale):              string(integreatlyv1alpha1.Version3Scale),
		string(integreatlyv1alpha1.ProductAMQOnline):           string(integreatlyv1alpha1.VersionAMQOnline),
		string(integreatlyv1alpha1.ProductApicurito):           string(integreatlyv1alpha1.VersionApicurito),
		string(integreatlyv1alpha1.ProductCodeReadyWorkspaces): string(integreatlyv1alpha1.VersionCodeReadyWorkspaces),
		string(integreatlyv1alpha1.ProductFuseOnOpenshift):     string(integreatlyv1alpha1.VersionFuseOnOpenshift),
		string(integreatlyv1alpha1.ProductUps):                 string(integreatlyv1alpha1.VersionUps),
		string(integreatlyv1alpha1.ProductRHSSOUser):           string(integreatlyv1alpha1.VersionRHSSOUser),
	}
	err = checkOperandVersions(t, f, namespace, integreatlyv1alpha1.ProductsStage, productOperands)
	if err != nil {
		return err
	}

	// check routes were created by checking hardcoded number of routes
	// would be nice if expected routes can be dynamically discovered
	expectedRoutes := map[string]int{
		"3scale":                         6,
		"amq-online":                     2,
		"apicurito":                      2,
		"codeready-workspaces":           3,
		"fuse":                           1,
		"middleware-monitoring-operator": 3,
		"rhsso":                          2,
		"solution-explorer":              1,
		"ups":                            1,
		"user-sso":                       2,
	}

	for product, numberRoutes := range expectedRoutes {
		err = checkRoutes(t, f, product, numberRoutes)
		if err != nil {
			return err
		}
	}

	// check no failed PVCs
	pvcNamespaces := []string{
		string(integreatlyv1alpha1.Product3Scale),
		string(integreatlyv1alpha1.ProductFuse),
		string(integreatlyv1alpha1.ProductRHSSO),
		string(integreatlyv1alpha1.ProductSolutionExplorer),
		string(integreatlyv1alpha1.ProductUps),
		string(integreatlyv1alpha1.ProductRHSSOUser),
	}
	err = checkPvcs(t, f, namespace, pvcNamespaces)
	return err
}

func checkIntegreatlyNamespaceLabels(t *testing.T, f *framework.Framework, namespaces []string, label string) error {
	for _, namespaceName := range namespaces {
		namespace := &corev1.Namespace{}
		err := f.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: intlyNamespacePrefix + namespaceName}, namespace)
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

func checkOperatorVersions(t *testing.T, f *framework.Framework, namespace string, stage integreatlyv1alpha1.StageName, operatorVersions map[string]string) error {
	installation := &integreatlyv1alpha1.RHMI{}

	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: common.InstallationName, Namespace: namespace}, installation)
	if err != nil {
		return fmt.Errorf("Error getting installation CR from cluster when checking operator versions: %w", err)
	}

	for product, version := range operatorVersions {
		clusterVersion := installation.Status.Stages[stage].Products[integreatlyv1alpha1.ProductName(product)].OperatorVersion
		if clusterVersion != integreatlyv1alpha1.OperatorVersion(version) {
			return fmt.Errorf("Error with version of %s operator deployed on cluster. Expected %s. Got %s", product, version, clusterVersion)
		}
	}

	return nil
}

func checkOperandVersions(t *testing.T, f *framework.Framework, namespace string, stage integreatlyv1alpha1.StageName, operandVersions map[string]string) error {
	installation := &integreatlyv1alpha1.RHMI{}

	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: common.InstallationName, Namespace: namespace}, installation)
	if err != nil {
		return fmt.Errorf("Error getting installation CR from cluster when checking operand versions: %w", err)
	}

	for product, version := range operandVersions {
		clusterVersion := installation.Status.Stages[stage].Products[integreatlyv1alpha1.ProductName(product)].Version
		if clusterVersion != integreatlyv1alpha1.ProductVersion(version) {
			return fmt.Errorf("Error with version of %s deployed on cluster. Expected %s. Got %s", product, version, clusterVersion)
		}
	}

	return nil
}

func checkRoutes(t *testing.T, f *framework.Framework, product string, numberRoutes int) error {
	routes := &routev1.RouteList{}
	err := f.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{Namespace: intlyNamespacePrefix + product})
	if err != nil {
		return fmt.Errorf("Error getting routes for %s namespace: %w", product, err)
	}
	if len(routes.Items) != numberRoutes {
		return fmt.Errorf("Expected %v routes in %v%v namespace. Found %v", numberRoutes, intlyNamespacePrefix, product, len(routes.Items))
	}
	return nil
}

func checkPvcs(t *testing.T, f *framework.Framework, s string, pvcNamespaces []string) error {
	for _, pvcNamespace := range pvcNamespaces {
		pvcs := &corev1.PersistentVolumeClaimList{}
		err := f.Client.List(goctx.TODO(), pvcs, &k8sclient.ListOptions{Namespace: intlyNamespacePrefix + pvcNamespace})
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
		installation := &integreatlyv1alpha1.RHMI{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: common.InstallationName, Namespace: namespace}, installation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s installation in namespace: %s, phase: %s\n", common.InstallationName, namespace, phase)
				return false, nil
			}
			return false, err
		}

		phaseStatus := fmt.Sprintf("%#v", installation.Status.Stages[integreatlyv1alpha1.StageName(phase)])
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

func IntegreatlyCluster(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) {
	namespace, err := ctx.GetNamespace()
	// Create SMTP Secret
	installationPrefix, found := os.LookupEnv("INSTALLATION_PREFIX")
	if !found {
		t.Fatal("INSTALLATION_PREFIX env var is not set")
	}

	var smtpSec = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprint(installationPrefix, "-smtp"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("test"),
			"password": []byte("test"),
			"port":     []byte("test"),
			"tls":      []byte("test"),
			"username": []byte("test"),
		},
	}
	err = f.Client.Create(context.TODO(), smtpSec, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	// create pagerduty secret
	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprint(installationPrefix, "-pagerduty"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"secretKey": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	err = f.Client.Create(context.TODO(), pagerdutySecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	// create dead mans snitch secret
	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprint(installationPrefix, "-deadmanssnitch"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"url": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	err = f.Client.Create(context.TODO(), dmsSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	// wait for integreatly-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "rhmi-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
	//TODO: split them into their own test cases
	// check that all of the operators deploy and all of the installation phases complete
	if err = integreatlyManagedTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}

	monitoringTimeout := 15 * time.Minute
	monitoringRetryInterval := 1 * time.Minute
	err = wait.Poll(monitoringRetryInterval, monitoringTimeout, func() (done bool, err error) {
		if err = integreatlyMonitoringTest(t, f, ctx); err != nil {
			t.Log("Waiting 1 minute for alerts to normalise before retrying integreatlyMonitoringTest")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = integreatlyGrafanaTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
