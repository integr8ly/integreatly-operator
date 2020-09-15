package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	appsv1 "github.com/openshift/api/apps/v1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type repeatFunc func()

type alertManagerConfig struct {
	Global struct {
		SMTPSmartHost    string `yaml:"smtp_smarthost"`
		SMTPAuthUsername string `yaml:"smtp_auth_username"`
		SMTPAuthPassword string `yaml:"smtp_auth_password"`
	} `yaml:"global"`

	Receivers []map[string]interface{} `yaml:"receivers"`
}

const (
	threescaleOperatorDeploymentName          = "3scale-operator"
	threescaleApicastProdDeploymentConfigName = "apicast-production"
	monitoringTimeout                         = time.Minute * 15
	monitoringRetryInterval                   = time.Minute
	verifyOperatorDeploymentTimeout           = time.Minute * 5
	verifyOperatorDeploymentRetryInterval     = time.Second * 15
)

var threescaleAlertsToTest = map[string]string{
	"RHMIThreeScaleApicastProductionServiceEndpointDown": "none",
	"ThreeScaleApicastProductionPod":                     "none",
}

// TestIntegreatlyAlertsMechanism verifies that alert mechanism works
func TestIntegreatlyAlertsMechanism(t *testing.T, ctx *TestingContext) {

	originalOperatorReplicas, err := getNumOfReplicasDeployment(threescaleOperatorDeploymentName, ctx.KubeClient)
	if err != nil {
		t.Errorf("failed to get number of replicas: %s", err)
	}

	// verify that alert to be tested is not firing before starting the test
	err = getThreescaleAlertState(ctx)
	if err != nil {
		t.Fatal("failed to get threescale alert state", err)
	}

	threescaleAlertsFiring := false

	for threescaleAlertName, threescaleAlertState := range threescaleAlertsToTest {
		if threescaleAlertState != "none" {
			threescaleAlertsFiring = true
			t.Errorf("%s alert should not be firing", threescaleAlertName)
		}
	}

	if threescaleAlertsFiring {
		t.FailNow()
	}

	// scale down Threescale operator and UI pods and verify that threescale alert is firing
	err = performTest(t, ctx, originalOperatorReplicas)
	if err != nil {
		t.Fatal(err)
	}

	// verify the operator has been scaled backup
	err = checkThreescaleOperatorReplicasAreReady(ctx, t, originalOperatorReplicas)
	if err != nil {
		t.Fatal(err)
	}

	// verify that threescale alert is not firing
	err = waitForThreescaleAlertState("none", ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	// verify alertmanager-application-monitoring secret
	err = verifySecrets(ctx.KubeClient)
	if err != nil {
		t.Fatal("failed to verify alertmanager-application-monitoring secret", err)
	}
}

func verifySecrets(kubeClient kubernetes.Interface) error {
	var pagerdutyKey, dmsURL string
	res, err := kubeClient.CoreV1().Secrets(RHMIOperatorNamespace).Get("redhat-rhmi-deadmanssnitch", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}
	if len(res.Data["SNITCH_URL"]) != 0 {
		dmsURL = string(res.Data["SNITCH_URL"])
	} else if len(res.Data["url"]) != 0 {
		dmsURL = string(res.Data["url"])
	} else {
		return fmt.Errorf("url is undefined in dead mans snitch secret")
	}

	res, err = kubeClient.CoreV1().Secrets(RHMIOperatorNamespace).Get("redhat-rhmi-pagerduty", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	if len(res.Data["PAGERDUTY_KEY"]) != 0 {
		pagerdutyKey = string(res.Data["PAGERDUTY_KEY"])
	} else if len(res.Data["serviceKey"]) != 0 {
		pagerdutyKey = string(res.Data["serviceKey"])
	} else {
		return fmt.Errorf("secret key is undefined in pager duty secret")
	}

	res, err = kubeClient.CoreV1().Secrets(RHMIOperatorNamespace).Get("redhat-rhmi-smtp", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}
	smtp := res.Data

	res, err = kubeClient.CoreV1().Secrets(MonitoringOperatorNamespace).Get("alertmanager-application-monitoring", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}
	monitoring := res.Data["alertmanager.yaml"]

	var config alertManagerConfig
	err = yaml.Unmarshal(monitoring, &config)
	if err != nil {
		return fmt.Errorf("failed to parse yaml: %w", err)
	}

	if config.Global.SMTPSmartHost != string(smtp["host"])+":"+string(smtp["port"]) {
		return fmt.Errorf("smtp_smarthost not set correctly")
	}
	if config.Global.SMTPAuthUsername != string(smtp["username"]) {
		return fmt.Errorf("smtp_auth_username not set correctly")
	}
	if config.Global.SMTPAuthPassword != string(smtp["password"]) {
		return fmt.Errorf("smtp_auth_password not set correctly")
	}

	for _, receiver := range config.Receivers {
		switch receiver["name"] {
		case "critical":
			configs := receiver["pagerduty_configs"].([]interface{})
			if configs[0].(map[interface{}]interface{})["service_key"] != pagerdutyKey {
				return fmt.Errorf("pagerduty service_key not set correctly")
			}
		case "deadmansswitch":
			configs := receiver["webhook_configs"].([]interface{})
			if configs[0].(map[interface{}]interface{})["url"] != dmsURL {
				return fmt.Errorf("dms url not set correctly")
			}
		}
	}

	return nil
}

func performTest(t *testing.T, ctx *TestingContext, originalOperatorReplicas int32) error {
	originalUIReplicas, err := getNumOfReplicasDeploymentConfig(threescaleApicastProdDeploymentConfigName, ThreeScaleProductNamespace, ctx.Client)
	if err != nil {
		t.Errorf("failed to get number of replicas: %s", err)
	}

	quit1 := make(chan struct{})
	go repeat(func() {
		scaleDeployment(threescaleOperatorDeploymentName, 0, ctx.KubeClient)
	}, quit1)
	defer close(quit1)
	defer scaleDeployment(threescaleOperatorDeploymentName, originalOperatorReplicas, ctx.KubeClient)

	quit2 := make(chan struct{})
	go repeat(func() {
		scaleDeploymentConfig(threescaleApicastProdDeploymentConfigName, ThreeScaleProductNamespace, 0, ctx.Client)
	}, quit2)
	defer close(quit2)
	defer scaleDeploymentConfig(threescaleApicastProdDeploymentConfigName, ThreeScaleProductNamespace, originalUIReplicas, ctx.Client)

	err = waitForThreescaleAlertState("pending", ctx, t)
	if err != nil {
		return err
	}

	err = waitForThreescaleAlertState("firing", ctx, t)
	if err != nil {
		return err
	}

	err = checkAlertManager(ctx, t)
	return err
}

func checkAlertManager(ctx *TestingContext, t *testing.T) error {
	output, err := execToPod("amtool alert --alertmanager.url=http://localhost:9093",
		"alertmanager-application-monitoring-0",
		MonitoringOperatorNamespace,
		"alertmanager",
		ctx)
	if err != nil {
		return fmt.Errorf("failed to exec to alertmanger pod: %w", err)
	}

	alertsNotFiringInAlertManager := false
	for threescaleAlertName := range threescaleAlertsToTest {
		if !strings.Contains(output, threescaleAlertName) {
			alertsNotFiringInAlertManager = true
			t.Errorf("%s alert not firing in alertmanager", threescaleAlertName)
		}
	}

	if alertsNotFiringInAlertManager {
		t.FailNow()
	}

	return nil
}

func repeat(function repeatFunc, quit chan struct{}) {
	for {
		select {
		case <-quit:
			return
		default:
			function()
		}
	}
}

func waitForThreescaleAlertState(expectedState string, ctx *TestingContext, t *testing.T) error {
	err := wait.PollImmediate(monitoringRetryInterval, monitoringTimeout, func() (done bool, err error) {
		err = getThreescaleAlertState(ctx)
		if err != nil {
			t.Log("failed to get threescale alert state:", err)
			t.Log("waiting 1 minute before retrying")
			return false, nil
		}

		alertsInExpectedState := true
		for threescaleAlertName, threescaleAlertState := range threescaleAlertsToTest {
			if threescaleAlertState != expectedState {
				alertsInExpectedState = false
				t.Logf("%s alert is not in expected state (%s) yet, current state: %s", threescaleAlertName, expectedState, threescaleAlertState)
				t.Log("waiting 1 minute before retrying")
			}
		}

		if alertsInExpectedState {
			return true, nil
		}

		return false, nil
	})

	return err
}

func getThreescaleAlertState(ctx *TestingContext) error {
	output, err := execToPod("curl localhost:9090/api/v1/alerts",
		"prometheus-application-monitoring-0",
		MonitoringOperatorNamespace,
		"prometheus",
		ctx)
	if err != nil {
		return fmt.Errorf("failed to exec to prometheus pod: %w", err)
	}

	var promAPICallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promAPICallOutput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	var alertsResult prometheusv1.AlertsResult
	err = json.Unmarshal(promAPICallOutput.Data, &alertsResult)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	for threescaleAlertName := range threescaleAlertsToTest {
		threescaleAlertsToTest[threescaleAlertName] = "none"
	}

	for _, alert := range alertsResult.Alerts {
		alertName := string(alert.Labels["alertname"])

		for threescaleAlertName := range threescaleAlertsToTest {
			if alertName == threescaleAlertName {
				threescaleAlertsToTest[threescaleAlertName] = string(alert.State)
			}
		}
	}

	return nil
}

func getNumOfReplicasDeployment(name string, kubeClient kubernetes.Interface) (int32, error) {
	deploymentsClient := kubeClient.AppsV1().Deployments(ThreeScaleOperatorNamespace)

	result, getErr := deploymentsClient.Get(name, metav1.GetOptions{})
	if getErr != nil {
		return 0, fmt.Errorf("failed to get latest version of Deployment: %v", getErr)
	}

	return *result.Spec.Replicas, nil
}

func getNumOfReplicasDeploymentConfig(name string, namespace string, client client.Client) (int32, error) {
	deploymentConfig := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	getErr := client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, deploymentConfig)
	if getErr != nil {
		return 0, fmt.Errorf("failed to get DeploymentConfig %s in namespace %s with error: %s", name, namespace, getErr)
	}

	return deploymentConfig.Spec.Replicas, nil
}

func scaleDeployment(name string, replicas int32, kubeClient kubernetes.Interface) error {
	deploymentsClient := kubeClient.AppsV1().Deployments(ThreeScaleOperatorNamespace)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := deploymentsClient.Get(name, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get latest version of Deployment: %v", getErr)
		}

		result.Spec.Replicas = &replicas
		_, updateErr := deploymentsClient.Update(result)
		return updateErr
	})
	if retryErr != nil {
		return fmt.Errorf("update failed: %v", retryErr)
	}

	return nil
}

func scaleDeploymentConfig(name string, namespace string, replicas int32, client client.Client) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		deploymentConfig := &appsv1.DeploymentConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		getErr := client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, deploymentConfig)
		if getErr != nil {
			return fmt.Errorf("failed to get DeploymentConfig %s in namespace %s with error: %s", name, namespace, getErr)
		}

		deploymentConfig.Spec.Replicas = replicas
		updateErr := client.Update(goctx.TODO(), deploymentConfig)
		return updateErr
	})
	if retryErr != nil {
		return fmt.Errorf("update failed: %v", retryErr)
	}

	return nil
}

func checkThreescaleOperatorReplicasAreReady(ctx *TestingContext, t *testing.T, originalOperatorReplicas int32) error {
	t.Logf("Checking correct number of threescale operator replicas (%d) are set", originalOperatorReplicas)
	err := wait.Poll(verifyOperatorDeploymentRetryInterval, verifyOperatorDeploymentTimeout, func() (done bool, err error) {
		numberOfOperatorReplicas, err := getNumOfReplicasDeployment(threescaleOperatorDeploymentName, ctx.KubeClient)

		if numberOfOperatorReplicas == originalOperatorReplicas {
			t.Log("Threescale operator deployment ready")
			return true, nil
		}

		if numberOfOperatorReplicas == 0 {
			t.Log("Threescale operator deployment not yet scaled, waiting 15 seconds before retrying")
			scaleDeployment(threescaleOperatorDeploymentName, originalOperatorReplicas, ctx.KubeClient)
			return false, nil
		}

		return false, err
	})

	if err != nil {
		return err
	}

	return nil
}
