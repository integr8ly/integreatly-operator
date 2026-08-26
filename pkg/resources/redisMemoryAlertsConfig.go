package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// RedisMemoryAlertsConfigMapName is the ConfigMap used to tune Redis memory
	// Prometheus alerts per installation. This follows the same pattern as the
	// rate-limit-alerts ConfigMap: the operator writes defaults once, and later
	// edits (by SRE / support) are preserved across reconciles.
	RedisMemoryAlertsConfigMapName = "redis-memory-alerts"
	RedisMemoryAlertsConfigMapKey  = "alerts"

	defaultRedisMemoryHighThreshold        = 80
	defaultRedisMemoryPredictThreshold     = 75
	defaultRedisMemoryPredictHours         = 5
	defaultRedisMemoryPredictDays          = 4
	defaultRedisMemoryPredictHoursLookback = "1h"
	defaultRedisMemoryPredictDaysLookback  = "6h"
	defaultRedisMemoryAlertSeverity        = "critical"
)

// RedisMemoryAlertsConfig is the JSON payload stored in redis-memory-alerts.
// Missing keys or zero-values fall back to the operator defaults so partial
// customer edits remain valid.
type RedisMemoryAlertsConfig struct {
	HighUsage   RedisMemoryUsageHighConfig `json:"RedisMemoryUsageHigh"`
	MaxIn4Hours RedisMemoryPredictConfig   `json:"RedisMemoryUsageMaxIn4Hours"`
	MaxIn4Days  RedisMemoryPredictConfig   `json:"RedisMemoryUsageMaxIn4Days"`
}

// RedisMemoryUsageHighConfig controls the static high-usage page.
// For noisy clusters that sit stably above the default 80% threshold, raise
// Threshold (e.g. 90 or 95) or lower Severity to "warning".
// For is the avg_over_time window (not the Prometheus pending duration), so a
// missed scrape does not reset the alert the way an instant vector + for: 30m does.
type RedisMemoryUsageHighConfig struct {
	Threshold int    `json:"threshold"`
	For       string `json:"for"`
	Severity  string `json:"severity"`
}

// RedisMemoryPredictConfig controls a predict_linear growth-rate alert.
// UsageThreshold is the floor at which the operator starts evaluating growth.
// PredictHours / PredictDays is the window in which usage is projected to hit 100%.
type RedisMemoryPredictConfig struct {
	UsageThreshold int    `json:"usageThreshold"`
	PredictHours   int    `json:"predictHours,omitempty"`
	PredictDays    int    `json:"predictDays,omitempty"`
	Lookback       string `json:"lookback"`
	For            string `json:"for"`
	Severity       string `json:"severity"`
}

// DefaultRedisMemoryAlertsConfig returns the in-tree defaults, matching current
// production MaxIn4Hours behaviour (75% gate, 5h prediction) so upgrade does not
// change paging. SRE can tighten per cluster via the ConfigMap (e.g. 80% / 4h).
func DefaultRedisMemoryAlertsConfig() RedisMemoryAlertsConfig {
	return RedisMemoryAlertsConfig{
		HighUsage: RedisMemoryUsageHighConfig{
			Threshold: defaultRedisMemoryHighThreshold,
			For:       alertFor30Mins,
			Severity:  defaultRedisMemoryAlertSeverity,
		},
		MaxIn4Hours: RedisMemoryPredictConfig{
			UsageThreshold: defaultRedisMemoryPredictThreshold,
			PredictHours:   defaultRedisMemoryPredictHours,
			Lookback:       defaultRedisMemoryPredictHoursLookback,
			For:            alertFor30Mins,
			Severity:       defaultRedisMemoryAlertSeverity,
		},
		MaxIn4Days: RedisMemoryPredictConfig{
			UsageThreshold: defaultRedisMemoryPredictThreshold,
			PredictDays:    defaultRedisMemoryPredictDays,
			Lookback:       defaultRedisMemoryPredictDaysLookback,
			For:            alertFor30Mins,
			Severity:       defaultRedisMemoryAlertSeverity,
		},
	}
}

func (c *RedisMemoryAlertsConfig) applyDefaults() {
	defaults := DefaultRedisMemoryAlertsConfig()
	c.HighUsage.applyDefaults(defaults.HighUsage)
	c.MaxIn4Hours.applyDefaults(defaults.MaxIn4Hours)
	c.MaxIn4Days.applyDefaults(defaults.MaxIn4Days)
}

func (c *RedisMemoryUsageHighConfig) applyDefaults(def RedisMemoryUsageHighConfig) {
	if c.Threshold <= 0 {
		c.Threshold = def.Threshold
	}
	c.For = validDurationOrDefault(c.For, def.For)
	if c.Severity == "" {
		c.Severity = def.Severity
	}
}

func (c *RedisMemoryPredictConfig) applyDefaults(def RedisMemoryPredictConfig) {
	if c.UsageThreshold <= 0 {
		c.UsageThreshold = def.UsageThreshold
	}
	if c.PredictHours <= 0 && def.PredictHours > 0 {
		c.PredictHours = def.PredictHours
	}
	if c.PredictDays <= 0 && def.PredictDays > 0 {
		c.PredictDays = def.PredictDays
	}
	c.Lookback = validDurationOrDefault(c.Lookback, def.Lookback)
	c.For = validDurationOrDefault(c.For, def.For)
	if c.Severity == "" {
		c.Severity = def.Severity
	}
}

// PromQL range / rule durations: integer + unit, optionally concatenated (1h30m).
// Units are ms, s, m, h, d, w, y. Go time.ParseDuration also accepts ns/us and
// negatives, which Prometheus rejects in range selectors.
var (
	promQLDurationRE = regexp.MustCompile(`^(?:[0-9]+(?:ms|[smhdwy]))+$`)
	promQLZeroRE     = regexp.MustCompile(`^(?:0+(?:ms|[smhdwy]))+$`)
)

func validDurationOrDefault(value, fallback string) string {
	if !isPositivePromQLDuration(value) {
		return fallback
	}
	return value
}

func isPositivePromQLDuration(value string) bool {
	return promQLDurationRE.MatchString(value) && !promQLZeroRE.MatchString(value)
}

// GetRedisMemoryAlertsConfig reads redis-memory-alerts from the RHMI operator
// namespace. A missing ConfigMap returns defaults so Redis reconcile does not
// fail if bootstrap has not created it yet.
func GetRedisMemoryAlertsConfig(ctx context.Context, client k8sclient.Client, namespace string) (RedisMemoryAlertsConfig, error) {
	cfg := DefaultRedisMemoryAlertsConfig()

	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      RedisMemoryAlertsConfigMapName,
		Namespace: namespace,
	}, configMap); err != nil {
		if k8serr.IsNotFound(err) {
			return cfg, nil
		}
		return cfg, err
	}

	raw, ok := configMap.Data[RedisMemoryAlertsConfigMapKey]
	if !ok || raw == "" {
		return cfg, nil
	}

	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultRedisMemoryAlertsConfig(), fmt.Errorf("failed to parse %s ConfigMap: %w", RedisMemoryAlertsConfigMapName, err)
	}
	cfg.applyDefaults()
	return cfg, nil
}
