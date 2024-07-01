package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/test/utils"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type alertManagerConfig struct {
	Global struct {
		SMTPSmartHost    string `yaml:"smtp_smarthost"`
		SMTPAuthUsername string `yaml:"smtp_auth_username"`
		SMTPAuthPassword string `yaml:"smtp_auth_password"`
	} `yaml:"global"`

	Receivers []map[string]interface{} `yaml:"receivers"`
}

const (
	keycloakOperatorDeploymentName        = "rhsso-operator"
	monitoringTimeout                     = time.Minute * 20
	monitoringRetryInterval               = time.Minute * 1
	verifyOperatorDeploymentTimeout       = time.Minute * 5
	verifyOperatorDeploymentRetryInterval = time.Second * 10
)

var keycloakAlertsToTest = map[string]string{
	"RHOAMRhssoKeycloakOperatorMetricsServiceEndpointDown": "none",
}

// TestIntegreatlyAlertsMechanism verifies that alert mechanism works
func TestIntegreatlyAlertsMechanism(t TestingTB, ctx *TestingContext) {

	originalOperatorReplicas, err := getNumOfReplicasDeployment(keycloakOperatorDeploymentName, ctx.KubeClient)
	if err != nil {
		t.Errorf("failed to get number of replicas: %s", err)
	}

	// verify that alert to be tested is not firing before starting the test
	err = getKeycloakAlertState(ctx)
	if err != nil {
		t.Fatal("failed to get keycloak alert state", err)
	}

	keycloakAlertsFiring := false

	for keycloakAlertName, keycloakAlertState := range keycloakAlertsToTest {
		if keycloakAlertState != "none" {
			keycloakAlertsFiring = true
			t.Errorf("%s alert should not be firing", keycloakAlertName)
		}
	}

	if keycloakAlertsFiring {
		t.Log("Keycloak alerts firing already, can not proceed with the test")
		t.FailNow()
	}

	t.Log("Keycloak alerts are not firing - scaling down keycloak operator deployment and performing tests")
	err = performTest(t, ctx, originalOperatorReplicas)
	if err != nil {
		t.Fatal("Error during testing keycloak operator alerts: %s", err)
	}

	// verify the operator has been scaled backup
	err = checkKeycloakOperatorReplicasAreReady(ctx, t, originalOperatorReplicas)
	if err != nil {
		t.Fatalf("Error During verify the operator has been scaled backup with error: %s", err)
	}
	t.Log("Keycloak operator deployment scaled back up and ready")

	// verify that keycloak alert is not firing
	err = waitForKeycloakAlertState("none", ctx, t)
	if err != nil {
		t.Fatal("Keycloak alerts failed to recover back to non-firing state with error: %s", err)
	}
	t.Log("Keycloak alerts are not firing")

	// verify alertmanager-rhoam secret
	err = verifySecrets(ctx.KubeClient)
	if err != nil {
		t.Fatal("failed to verify %s secret with err: %s", config.AlertManagerConfigSecretName, err)
	}

	t.Log("Alert mechanism test successful")
}

func verifySecrets(kubeClient kubernetes.Interface) error {
	var pagerdutyKey, dmsURL string
	dmsSecretFound := true
	res, err := kubeClient.CoreV1().Secrets(RHOAMOperatorNamespace).Get(goctx.TODO(), NamespacePrefix+"deadmanssnitch", metav1.GetOptions{})
	if err != nil {
		dmsSecretFound = false
	}
	if dmsSecretFound {
		if len(res.Data["SNITCH_URL"]) != 0 {
			dmsURL = string(res.Data["SNITCH_URL"])
		} else if len(res.Data["url"]) != 0 {
			dmsURL = string(res.Data["url"])
		} else {
			return fmt.Errorf("url is undefined in dead mans snitch secret")
		}
	}

	res, err = kubeClient.CoreV1().Secrets(RHOAMOperatorNamespace).Get(goctx.TODO(), NamespacePrefix+"pagerduty", metav1.GetOptions{})
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

	smtp, err := utils.GetSMTPSecret(kubeClient, RHOAMOperatorNamespace, SMTPSecretName)
	if err != nil {
		return err
	}

	res, err = kubeClient.CoreV1().Secrets(ObservabilityProductNamespace).Get(goctx.TODO(), config.AlertManagerConfigSecretName, metav1.GetOptions{})
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
			if dmsSecretFound {
				configs := receiver["webhook_configs"].([]interface{})
				if configs[0].(map[interface{}]interface{})["url"] != dmsURL {
					return fmt.Errorf("dms url not set correctly")
				}
			}
		}
	}

	return nil
}

func performTest(t TestingTB, ctx *TestingContext, originalOperatorReplicas int32) error {
	scaleDeployment(t, keycloakOperatorDeploymentName, 0, ctx.KubeClient)

	defer scaleDeployment(t, keycloakOperatorDeploymentName, originalOperatorReplicas, ctx.KubeClient)

	err := waitForKeycloakAlertState("firing", ctx, t)
	if err != nil {
		return err
	}
	err = checkAlertManager(ctx, t)
	return err
}

func checkAlertManager(ctx *TestingContext, t TestingTB) error {
	output, err := execToPod("amtool alert --alertmanager.url=http://localhost:9093",
		"alertmanager-rhoam-0",
		ObservabilityProductNamespace,
		"alertmanager",
		ctx)
	if err != nil {
		return fmt.Errorf("failed to exec to alertmanger pod: %w", err)
	}

	alertsNotFiringInAlertManager := false
	for keycloakAlertName := range keycloakAlertsToTest {
		if !strings.Contains(output, keycloakAlertName) {
			alertsNotFiringInAlertManager = true
			t.Errorf("%s alert not firing in alertmanager", keycloakAlertName)
		}
	}

	if alertsNotFiringInAlertManager {
		t.FailNow()
	}

	return nil
}

func waitForKeycloakAlertState(expectedState string, ctx *TestingContext, t TestingTB) error {
	err := wait.PollUntilContextTimeout(goctx.TODO(), monitoringRetryInterval, monitoringTimeout, true, func(ctx2 goctx.Context) (done bool, err error) {
		err = getKeycloakAlertState(ctx)
		if err != nil {
			t.Log("failed to get keycloak alert state:", err)
			t.Log("waiting 1 minute before retrying")
			return false, nil
		}

		alertsInExpectedState := true
		for keycloakAlertName, keycloakAlertState := range keycloakAlertsToTest {
			if keycloakAlertState != expectedState {
				alertsInExpectedState = false
				t.Logf("%s alert is not in expected state (%s) yet, current state: %s", keycloakAlertName, expectedState, keycloakAlertState)
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

func getKeycloakAlertState(ctx *TestingContext) error {
	output, err := execToPod("wget -qO - localhost:9090/api/v1/alerts",
		ObservabilityPrometheusPodName,
		ObservabilityProductNamespace,
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

	for keycloakAlertName := range keycloakAlertsToTest {
		keycloakAlertsToTest[keycloakAlertName] = "none"
	}

	for _, alert := range alertsResult.Alerts {
		alertName := string(alert.Labels["alertname"])

		for keycloakAlertName := range keycloakAlertsToTest {
			if alertName == keycloakAlertName {
				keycloakAlertsToTest[keycloakAlertName] = string(alert.State)
			}
		}
	}

	return nil
}

func getNumOfReplicasDeployment(name string, kubeClient kubernetes.Interface) (int32, error) {
	deploymentsClient := kubeClient.AppsV1().Deployments(RHSSOOperatorNamespace)

	result, getErr := deploymentsClient.Get(goctx.TODO(), name, metav1.GetOptions{})
	if getErr != nil {
		return 0, fmt.Errorf("failed to get latest version of Deployment: %v", getErr)
	}

	return *result.Spec.Replicas, nil
}

func scaleDeployment(t TestingTB, name string, replicas int32, kubeClient kubernetes.Interface) {
	deploymentsClient := kubeClient.AppsV1().Deployments(RHSSOOperatorNamespace)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := deploymentsClient.Get(goctx.TODO(), name, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get latest version of Deployment: %v", getErr)
		}

		result.Spec.Replicas = &replicas
		_, updateErr := deploymentsClient.Update(goctx.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		t.Logf("update failed: %v", retryErr)
	}

}

func checkKeycloakOperatorReplicasAreReady(ctx *TestingContext, t TestingTB, originalOperatorReplicas int32) error {
	t.Logf("Checking correct number of keycloak operator replicas (%d) are set", originalOperatorReplicas)
	err := wait.PollUntilContextTimeout(goctx.TODO(), verifyOperatorDeploymentRetryInterval, verifyOperatorDeploymentTimeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		numberOfOperatorReplicas, err := getNumOfReplicasDeployment(keycloakOperatorDeploymentName, ctx.KubeClient)

		if numberOfOperatorReplicas == originalOperatorReplicas {
			t.Log("Keycloak operator deployment ready")
			return true, nil
		}

		if numberOfOperatorReplicas == 0 {
			t.Log("Keycloak operator deployment not yet scaled, waiting 15 seconds before retrying")
			scaleDeployment(t, keycloakOperatorDeploymentName, originalOperatorReplicas, ctx.KubeClient)
			return false, nil
		}

		return false, err
	})

	if err != nil {
		return err
	}

	return nil
}
