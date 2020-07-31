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
	fuseOperatorDeploymentName            = "syndesis-operator"
	fuseUIDeploymentConfigName            = "syndesis-ui"
	monitoringTimeout                     = time.Minute * 15
	monitoringRetryInterval               = time.Minute
	verifyOperatorDeploymentTimeout       = time.Minute * 5
	verifyOperatorDeploymentRetryInterval = time.Second * 15
)

var fuseAlertsToTest = map[string]string{
	"FuseOnlineSyndesisUIInstanceDown":            "none",
	"RHMIFuseOnlineSyndesisUiServiceEndpointDown": "none",
}

// TestIntegreatlyAlertsMechanism verifies that alert mechanism works
func TestIntegreatlyAlertsMechanism(t *testing.T, ctx *TestingContext) {

	originalOperatorReplicas, err := getNumOfReplicasDeployment(fuseOperatorDeploymentName, ctx.KubeClient)
	if err != nil {
		t.Errorf("failed to get number of replicas: %s", err)
	}

	// verify that alert to be tested is not firing before starting the test
	err = getFuseAlertState(ctx)
	if err != nil {
		t.Fatal("failed to get fuse alert state", err)
	}

	fuseAlertsFiring := false

	for fuseAlertName, fuseAlertState := range fuseAlertsToTest {
		if fuseAlertState != "none" {
			fuseAlertsFiring = true
			t.Errorf("%s alert should not be firing", fuseAlertName)
		}
	}

	if fuseAlertsFiring {
		t.FailNow()
	}

	// scale down Fuse operator and UI pods and verify that fuse alert is firing
	err = performTest(t, ctx, originalOperatorReplicas)
	if err != nil {
		t.Fatal(err)
	}

	// verify the operator has been scaled backup
	err = checkFuseOperatorReplicasAreReady(ctx, t, originalOperatorReplicas)
	if err != nil {
		t.Fatal(err)
	}

	// verify that fuse alert is not firing
	err = waitForFuseAlertState("none", ctx, t)
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
	var pagerdutyKey string
	res, err := kubeClient.CoreV1().Secrets(RHMIOperatorNamespace).Get("redhat-rhmi-deadmanssnitch", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}
	dms := string(res.Data["url"])

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
			if configs[0].(map[interface{}]interface{})["url"] != dms {
				return fmt.Errorf("dms url not set correctly")
			}
		}
	}

	return nil
}

func performTest(t *testing.T, ctx *TestingContext, originalOperatorReplicas int32) error {
	originalUIReplicas, err := getNumOfReplicasDeploymentConfig(fuseUIDeploymentConfigName, FuseProductNamespace, ctx.Client)
	if err != nil {
		t.Errorf("failed to get number of replicas: %s", err)
	}

	quit1 := make(chan struct{})
	go repeat(func() {
		scaleDeployment(fuseOperatorDeploymentName, 0, ctx.KubeClient)
	}, quit1)
	defer close(quit1)
	defer scaleDeployment(fuseOperatorDeploymentName, originalOperatorReplicas, ctx.KubeClient)

	quit2 := make(chan struct{})
	go repeat(func() {
		scaleDeploymentConfig(fuseUIDeploymentConfigName, FuseProductNamespace, 0, ctx.Client)
	}, quit2)
	defer close(quit2)
	defer scaleDeploymentConfig(fuseUIDeploymentConfigName, FuseProductNamespace, originalUIReplicas, ctx.Client)

	err = waitForFuseAlertState("pending", ctx, t)
	if err != nil {
		return err
	}

	err = waitForFuseAlertState("firing", ctx, t)
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
	for fuseAlertName := range fuseAlertsToTest {
		if !strings.Contains(output, fuseAlertName) {
			alertsNotFiringInAlertManager = true
			t.Errorf("%s alert not firing in alertmanager", fuseAlertName)
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

func waitForFuseAlertState(expectedState string, ctx *TestingContext, t *testing.T) error {
	err := wait.PollImmediate(monitoringRetryInterval, monitoringTimeout, func() (done bool, err error) {
		err = getFuseAlertState(ctx)
		if err != nil {
			t.Log("failed to get fuse alert state:", err)
			t.Log("waiting 1 minute before retrying")
			return false, nil
		}

		alertsInExpectedState := true
		for fuseAlertName, fuseAlertState := range fuseAlertsToTest {
			if fuseAlertState != expectedState {
				alertsInExpectedState = false
				t.Logf("%s alert is not in expected state (%s) yet, current state: %s", fuseAlertName, expectedState, fuseAlertState)
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

func getFuseAlertState(ctx *TestingContext) error {
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

	for fuseAlertName := range fuseAlertsToTest {
		fuseAlertsToTest[fuseAlertName] = "none"
	}

	for _, alert := range alertsResult.Alerts {
		alertName := string(alert.Labels["alertname"])

		for fuseAlertName := range fuseAlertsToTest {
			if alertName == fuseAlertName {
				fuseAlertsToTest[fuseAlertName] = string(alert.State)
			}
		}
	}

	return nil
}

func getNumOfReplicasDeployment(name string, kubeClient kubernetes.Interface) (int32, error) {
	deploymentsClient := kubeClient.AppsV1().Deployments(FuseOperatorNamespace)

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
	deploymentsClient := kubeClient.AppsV1().Deployments(FuseOperatorNamespace)

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

func checkFuseOperatorReplicasAreReady(ctx *TestingContext, t *testing.T, originalOperatorReplicas int32) error {
	t.Logf("Checking correct number of fuse operator replicas (%d) are set", originalOperatorReplicas)
	err := wait.Poll(verifyOperatorDeploymentRetryInterval, verifyOperatorDeploymentTimeout, func() (done bool, err error) {
		numberOfOperatorReplicas, err := getNumOfReplicasDeployment(fuseOperatorDeploymentName, ctx.KubeClient)

		if numberOfOperatorReplicas == originalOperatorReplicas {
			t.Log("Fuse operator deployment ready")
			return true, nil
		}

		if numberOfOperatorReplicas == 0 {
			t.Log("Fuse operator deployment not yet scaled, waiting 15 seconds before retrying")
			scaleDeployment(fuseOperatorDeploymentName, originalOperatorReplicas, ctx.KubeClient)
			return false, nil
		}

		return false, err
	})

	if err != nil {
		return err
	}

	return nil
}
