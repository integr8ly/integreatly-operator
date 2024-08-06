package common

import (
	"context"
	"fmt"
	k8sappsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Alert struct {
	Labels struct {
		AlertName string `json:"alertname"`
	} `json:"labels"`
	State string `json:"state"`
}

type AlertsResults struct {
	Alerts []Alert `json:"alerts"`
}

func TestThreeScaleUIBBTAlerts(t TestingTB, ctx *TestingContext) {

	// Get the kube client and namespace from the shared test context
	client := ctx.Client

	// Scale down 3scale-operator to 0 pods
	err := scaleDeployments(context.TODO(), t, client, ThreeScaleOperatorNamespace, "threescale-operator-controller-manager-v2", 0)

	defer func() {
		err := scaleDeployments(context.TODO(), t, client, ThreeScaleOperatorNamespace, "threescale-operator-controller-manager-v2", 1)
		if err != nil {
			t.Logf("Failed to scale up 3scale-operator: %v", err)
		}

		t.Logf("Deployment successfully scaled back up")

		// Poll until alerts will stop firing

		err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Minute, 7*time.Minute, false, func(ctx2 context.Context) (bool, error) {
			// Check condition here
			t.Logf("Checking if alerts stopped firing. Repeating for 7 minutes, every 1 minute")
			isFiring, err := isThreeScaleUIBBTAlertFiring(ctx, t)
			if err != nil {
				// If there's an error, return it to stop polling
				return false, err
			}
			// If the alert is still firing, continue polling
			return !isFiring, nil
		})

		if err != nil {
			t.Logf("Failed to check if ThreeScale**UIBBT alert stopped firing: %v", err)
		} else {
			t.Logf("ThreeScale**UIBBT alert successfully stopped firing")
		}

	}()

	if err != nil {
		t.Fatalf("Failed to scale down 3scale-operator: %v", err)
	}

	time.Sleep(1 * time.Minute)

	// Scale down system-app to 0 pods
	err = scaleDeployments(context.TODO(), t, client, ThreeScaleProductNamespace, "system-app", 0)

	defer func() {
		err = scaleDeployments(context.TODO(), t, client, ThreeScaleProductNamespace, "system-app", 1)
		if err != nil {
			t.Logf("Failed to scale up system-app: %v", err)
		}

		t.Logf("Deployment successfully scaled back up")

	}()

	if err != nil {
		t.Fatalf("Failed to scale down system-app: %v", err)
	}

	err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Minute, 7*time.Minute, false, func(ctx2 context.Context) (bool, error) {
		// Check if alerts are firing
		t.Logf("Checking if alerts started firing. Repeating for 7 minutes, every 1 minute")
		isFiring, err := isThreeScaleUIBBTAlertFiring(ctx, t)
		if err != nil {
			// If there's an error, return it to stop polling
			return false, err
		}
		// If the alert is firing, stop polling
		return isFiring, nil
	})

	if err != nil {
		t.Fatalf("Failed to check if ThreeScale**UIBBT alert is firing: %v", err)
	}

	// Check if alert is firing after polling
	isThreeScaleUIBBTAlertFiring, err := isThreeScaleUIBBTAlertFiring(ctx, t)
	if err != nil {
		t.Fatalf("Failed to check if ThreeScale**UIBBT alert is firing: %v", err)
	}
	if !isThreeScaleUIBBTAlertFiring {
		t.Fatalf("ThreeScale**UIBBT alert is not firing")
	}

}

func scaleDeployments(ctx context.Context, t TestingTB, client k8sclient.Client, namespace string, deploymentName string, replicas int32) error {

	deployment := &k8sappsv1.Deployment{}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{
		Name:      deploymentName,
		Namespace: namespace,
	}, deployment)

	if err != nil {
		return fmt.Errorf("failed to get deployment %q: %v ", deployment.Name, err)
	}

	deployment.Spec.Replicas = &replicas

	t.Logf("Scaling deployment")

	err = client.Update(ctx, deployment)

	if err != nil {
		return fmt.Errorf("failed to update Deployment object %v", err)
	}
	t.Logf("Scaled deployment %q in namespace %q\n", deploymentName, namespace)

	return nil
}

func isThreeScaleUIBBTAlertFiring(ctx *TestingContext, t TestingTB) (bool, error) {

	alertNames := map[string]bool{
		"ThreeScaleAdminUIBBT":       false,
		"ThreeScaleDeveloperUIBBT":   false,
		"ThreeScaleSystemAdminUIBBT": false,
	}

	output, err := execToPod("wget -qO - localhost:9090/api/v1/alerts",
		ObservabilityPrometheusPodName,
		ObservabilityProductNamespace,
		"prometheus",
		ctx)
	if err != nil {
		return false, fmt.Errorf("failed to exec to prometheus pod: %w", err)
	}

	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	var alertsResult AlertsResults
	err = json.Unmarshal(promApiCallOutput.Data, &alertsResult)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	for _, alert := range alertsResult.Alerts {
		if _, ok := alertNames[alert.Labels.AlertName]; ok && alert.State == "firing" {
			alertNames[alert.Labels.AlertName] = true
		}
	}

	for _, firing := range alertNames {
		if !firing {
			// If any of the alerts are not firing, return false
			return false, nil
		}
	}

	// If we got here, all alerts are firing
	t.Logf("alerts are firing")
	return true, nil
}
