package config

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RateLimitConfigMapName = "sku-limits-managed-api-service"
	AlertConfigMapName     = "rate-limit-alerts"
	ManagedApiServiceSKU   = "RHOAM SERVICE SKU"

	DefaultRateLimitUnit     = "minute"
	DefaultRateLimitRequests = 13860
)

type RateLimitConfig struct {
	Unit            string `json:"unit"`
	RequestsPerUnit uint32 `json:"requests_per_unit"`
}

type AlertConfig struct {
	RuleName string  `json:"ruleName"`
	Level    string  `json:"level"`
	MinRate  string  `json:"minRate"`
	MaxRate  *string `json:"maxRate,omitempty"`
	Period   string  `json:"period"`
}

// GetRateLimitConfig retrieves the configuration for the rate limit service,
// taken from a ConfigMap that is expected to exist in the managed api operator
// namespace.
func GetRateLimitConfig(ctx context.Context, client k8sclient.Client, namespace string) (*RateLimitConfig, error) {
	skuConfigs := map[string]*RateLimitConfig{}
	if err := getFromJSONConfigMap(
		ctx, client,
		RateLimitConfigMapName, namespace, "rate_limit",
		&skuConfigs,
	); err != nil {
		return nil, err
	}

	sku, err := GetSKU(ctx, client)
	if err != nil {
		return nil, err
	}

	if result, ok := skuConfigs[sku]; ok {
		return result, nil
	}

	return nil, fmt.Errorf("SKU %s not found in ConfigMap", sku)
}

func GetAlertConfig(ctx context.Context, client k8sclient.Client, namespace string) (map[string]*AlertConfig, error) {
	alertsConfig := map[string]*AlertConfig{}
	err := getFromJSONConfigMap(
		ctx, client,
		AlertConfigMapName, namespace, "alerts",
		&alertsConfig,
	)

	return alertsConfig, err
}

func GetSKU(_ context.Context, _ k8sclient.Client) (string, error) {
	return ManagedApiServiceSKU, nil
}

func getFromJSONConfigMap(ctx context.Context, client k8sclient.Client, cmName, namespace, configkey string, v interface{}) error {
	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      cmName,
		Namespace: namespace,
	}, configMap); err != nil {
		return err
	}

	configJSON, ok := configMap.Data[configkey]
	if !ok {
		return fmt.Errorf("%s not found in %s ConfigMap data", configkey, cmName)
	}

	return json.Unmarshal([]byte(configJSON), v)
}
