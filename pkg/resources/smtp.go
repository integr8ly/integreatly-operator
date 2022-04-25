package resources

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	alertManagerConfigSecretFileName = "alertmanager.yaml"
	alertManagerConfigSecretName     = "alertmanager-application-monitoring"
)

type alertManagerConfig struct {
	Global map[string]string `yaml:"global"`
}

func GetExistingSMTPFromAddress(ctx context.Context, client k8sclient.Client, ns string) (string, error) {
	amSecret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      alertManagerConfigSecretName,
		Namespace: ns,
	}, amSecret)
	if err != nil {
		return "", err
	}
	monitoring := amSecret.Data[alertManagerConfigSecretFileName]
	var config alertManagerConfig
	err = yaml.Unmarshal(monitoring, &config)
	if err != nil {
		return "", fmt.Errorf("failed to parse alert monitoring yaml: %w", err)
	}

	return config.Global["smtp_from"], nil
}
