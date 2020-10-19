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
	ManagedApiServiceSKU   = "RHOAM SERVICE SKU"

	DefaultRateLimitUnit     = "minute"
	DefaultRateLimitRequests = 13860
)

type RateLimitConfig struct {
	Unit            string `json:"unit"`
	RequestsPerUnit uint32 `json:"requests_per_unit"`
}

type AlertConfig struct {
	Name    string `json:"name"`
	Level   string `json:"level"`
	MinRate string `json:"minRate"`
	MaxRate string `json:"maxRate"`
	Period  string `json:"period"`
}

// GetRateLimitConfig retrieves the configuration for the rate limit service,
// taken from a ConfigMap that is expected to exist in the managed api operator
// namespace.
func GetRateLimitConfig(ctx context.Context, client k8sclient.Client, namespace string) (*RateLimitConfig, error) {
	sku, err := GetSKU(ctx, client)
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      RateLimitConfigMapName,
		Namespace: namespace,
	}, configMap); err != nil {
		return nil, err
	}

	skuConfigsJSON, ok := configMap.Data["rate_limit"]
	if !ok {
		return nil, fmt.Errorf("rate_limit key not found in config map")
	}

	skuConfigs := map[string]*RateLimitConfig{}
	if err := json.Unmarshal([]byte(skuConfigsJSON), &skuConfigs); err != nil {
		return nil, err
	}

	if result, ok := skuConfigs[sku]; ok {
		return result, nil
	}

	return nil, fmt.Errorf("SKU %s not found in ConfigMap", sku)
}

func GetAlertConfig(ctx context.Context, client k8sclient.Client) ([]*AlertConfig, error) {
	// TODO
	return nil, nil
}

func GetSKU(_ context.Context, _ k8sclient.Client) (string, error) {
	return ManagedApiServiceSKU, nil
}
