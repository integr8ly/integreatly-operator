package resources

import (
	"context"
	"fmt"
	"testing"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetRedisMemoryAlertsConfig(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	namespace := "redhat-rhoam-operator"

	t.Run("missing configmap returns defaults", func(t *testing.T) {
		client := utils.NewTestClient(scheme)
		cfg, err := GetRedisMemoryAlertsConfig(context.TODO(), client, namespace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defaults := DefaultRedisMemoryAlertsConfig()
		if cfg.HighUsage.Threshold != defaults.HighUsage.Threshold {
			t.Errorf("threshold = %d, want %d", cfg.HighUsage.Threshold, defaults.HighUsage.Threshold)
		}
		if cfg.MaxIn4Hours.PredictHours != defaults.MaxIn4Hours.PredictHours {
			t.Errorf("predictHours = %d, want %d", cfg.MaxIn4Hours.PredictHours, defaults.MaxIn4Hours.PredictHours)
		}
	})

	t.Run("partial config fills remaining defaults", func(t *testing.T) {
		raw := `{
			"RedisMemoryUsageHigh": {
				"threshold": 90,
				"severity": "warning"
			}
		}`
		client := utils.NewTestClient(scheme, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RedisMemoryAlertsConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				RedisMemoryAlertsConfigMapKey: raw,
			},
		})

		cfg, err := GetRedisMemoryAlertsConfig(context.TODO(), client, namespace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HighUsage.Threshold != 90 {
			t.Errorf("threshold = %d, want 90", cfg.HighUsage.Threshold)
		}
		if cfg.HighUsage.Severity != "warning" {
			t.Errorf("severity = %s, want warning", cfg.HighUsage.Severity)
		}
		if cfg.HighUsage.For != alertFor30Mins {
			t.Errorf("for = %s, want %s", cfg.HighUsage.For, alertFor30Mins)
		}
		if cfg.MaxIn4Hours.UsageThreshold != defaultRedisMemoryPredictThreshold {
			t.Errorf("max in 4 hours usage threshold = %d, want %d", cfg.MaxIn4Hours.UsageThreshold, defaultRedisMemoryPredictThreshold)
		}
		if cfg.MaxIn4Hours.PredictHours != defaultRedisMemoryPredictHours {
			t.Errorf("predictHours = %d, want %d", cfg.MaxIn4Hours.PredictHours, defaultRedisMemoryPredictHours)
		}
	})

	t.Run("invalid duration falls back to default", func(t *testing.T) {
		raw := `{
			"RedisMemoryUsageHigh": {
				"threshold": 95,
				"for": "not-a-duration"
			}
		}`
		client := utils.NewTestClient(scheme, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RedisMemoryAlertsConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				RedisMemoryAlertsConfigMapKey: raw,
			},
		})

		cfg, err := GetRedisMemoryAlertsConfig(context.TODO(), client, namespace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HighUsage.For != alertFor30Mins {
			t.Errorf("for = %s, want %s", cfg.HighUsage.For, alertFor30Mins)
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		client := utils.NewTestClient(scheme, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RedisMemoryAlertsConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				RedisMemoryAlertsConfigMapKey: "{not-json",
			},
		})

		_, err := GetRedisMemoryAlertsConfig(context.TODO(), client, namespace)
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})

	t.Run("missing alerts key returns defaults", func(t *testing.T) {
		client := utils.NewTestClient(scheme, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RedisMemoryAlertsConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{},
		})
		cfg, err := GetRedisMemoryAlertsConfig(context.TODO(), client, namespace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HighUsage.Threshold != defaultRedisMemoryHighThreshold {
			t.Errorf("threshold = %d, want %d", cfg.HighUsage.Threshold, defaultRedisMemoryHighThreshold)
		}
	})

	t.Run("zero values are filled from defaults", func(t *testing.T) {
		raw := `{
			"RedisMemoryUsageHigh": {
				"threshold": 0,
				"severity": ""
			},
			"RedisMemoryUsageMaxIn4Hours": {
				"usageThreshold": 0,
				"predictHours": 0,
				"lookback": "",
				"for": "",
				"severity": ""
			},
			"RedisMemoryUsageMaxIn4Days": {
				"usageThreshold": 0,
				"predictDays": 0
			}
		}`
		client := utils.NewTestClient(scheme, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RedisMemoryAlertsConfigMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				RedisMemoryAlertsConfigMapKey: raw,
			},
		})
		cfg, err := GetRedisMemoryAlertsConfig(context.TODO(), client, namespace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HighUsage.Threshold != defaultRedisMemoryHighThreshold {
			t.Errorf("high threshold = %d, want %d", cfg.HighUsage.Threshold, defaultRedisMemoryHighThreshold)
		}
		if cfg.HighUsage.Severity != defaultRedisMemoryAlertSeverity {
			t.Errorf("high severity = %s, want %s", cfg.HighUsage.Severity, defaultRedisMemoryAlertSeverity)
		}
		if cfg.MaxIn4Hours.UsageThreshold != defaultRedisMemoryPredictThreshold {
			t.Errorf("hours usage threshold = %d, want %d", cfg.MaxIn4Hours.UsageThreshold, defaultRedisMemoryPredictThreshold)
		}
		if cfg.MaxIn4Hours.PredictHours != defaultRedisMemoryPredictHours {
			t.Errorf("predictHours = %d, want %d", cfg.MaxIn4Hours.PredictHours, defaultRedisMemoryPredictHours)
		}
		if cfg.MaxIn4Hours.Severity != defaultRedisMemoryAlertSeverity {
			t.Errorf("hours severity = %s, want %s", cfg.MaxIn4Hours.Severity, defaultRedisMemoryAlertSeverity)
		}
		if cfg.MaxIn4Days.PredictDays != defaultRedisMemoryPredictDays {
			t.Errorf("predictDays = %d, want %d", cfg.MaxIn4Days.PredictDays, defaultRedisMemoryPredictDays)
		}
	})

	t.Run("get error is returned", func(t *testing.T) {
		mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
		mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
			return fmt.Errorf("generic error")
		}
		_, err := GetRedisMemoryAlertsConfig(context.TODO(), mockClient, namespace)
		if err == nil {
			t.Fatal("expected error from client.Get")
		}
	})
}

func TestValidDurationOrDefault(t *testing.T) {
	tests := []struct {
		value    string
		fallback string
		want     string
	}{
		{value: "", fallback: "30m", want: "30m"},
		{value: "1h", fallback: "30m", want: "1h"},
		{value: "1h30m", fallback: "30m", want: "1h30m"},
		{value: "1d", fallback: "30m", want: "1d"},
		{value: "1w", fallback: "30m", want: "1w"},
		{value: "1us", fallback: "30m", want: "30m"},
		{value: "1ns", fallback: "30m", want: "30m"},
		{value: "-1h", fallback: "30m", want: "30m"},
		{value: "0s", fallback: "30m", want: "30m"},
		{value: "not-a-duration", fallback: "30m", want: "30m"},
	}
	for _, tt := range tests {
		if got := validDurationOrDefault(tt.value, tt.fallback); got != tt.want {
			t.Errorf("validDurationOrDefault(%q, %q) = %q, want %q", tt.value, tt.fallback, got, tt.want)
		}
	}
}
