package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	crov1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateRedisMemoryUsageAlerts(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	inst := &v1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhoam",
			Namespace: "redhat-rhoam-operator",
		},
		Spec: v1alpha1.RHMISpec{
			UseClusterStorage: "false",
		},
	}
	redis := &crov1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend-redis",
			Namespace: inst.Namespace,
			Labels: map[string]string{
				"productName": "3scale",
			},
		},
	}
	ruleNs := config.GetOboNamespace(inst.Namespace)
	log := l.NewLogger()

	t.Run("defaults match current production predict windows", func(t *testing.T) {
		client := utils.NewTestClient(scheme)
		if err := createRedisMemoryUsageAlerts(context.TODO(), client, inst, redis, log); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertPrometheusRule(t, client, "redis-memory-usage-high", ruleNs, "RedisMemoryUsageHigh", "critical", redisMemoryUsageHighPending,
			"avg_over_time(cro_redis_memory_usage_percentage_average[30m]) > 80")
		assertPrometheusRule(t, client, "redis-memory-usage-will-max-in-4-hours", ruleNs, "RedisMemoryUsageMaxIn4Hours", "critical", alertFor30Mins,
			"(predict_linear(sum by (instanceID) (cro_redis_memory_usage_percentage_average{job='operator-metrics-service'})[1h:1m], 5 * 3600) >= 100) and on (instanceID) (cro_redis_memory_usage_percentage_average{job='operator-metrics-service'} > 75)")
		assertPrometheusRule(t, client, "redis-memory-usage-max-fill-in-4-days", ruleNs, "RedisMemoryUsageMaxIn4Days", "critical", alertFor30Mins,
			"(predict_linear(sum by (instanceID) (cro_redis_memory_usage_percentage_average{job='operator-metrics-service'})[6h:1m], 4 * 24 * 3600) >= 100) and on (instanceID) (cro_redis_memory_usage_percentage_average{job='operator-metrics-service'} > 75)")
	})

	t.Run("configmap overrides threshold severity and growth window", func(t *testing.T) {
		cfg := DefaultRedisMemoryAlertsConfig()
		cfg.HighUsage.Threshold = 90
		cfg.HighUsage.Severity = "warning"
		cfg.HighUsage.For = "1h"
		cfg.MaxIn4Hours.UsageThreshold = 85
		cfg.MaxIn4Hours.PredictHours = 6
		cfg.MaxIn4Hours.Lookback = "2h"
		raw, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		client := utils.NewTestClient(scheme, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RedisMemoryAlertsConfigMapName,
				Namespace: inst.Namespace,
			},
			Data: map[string]string{
				RedisMemoryAlertsConfigMapKey: string(raw),
			},
		})

		if err := createRedisMemoryUsageAlerts(context.TODO(), client, inst, redis, log); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assertPrometheusRule(t, client, "redis-memory-usage-high", ruleNs, "RedisMemoryUsageHigh", "warning", redisMemoryUsageHighPending,
			"avg_over_time(cro_redis_memory_usage_percentage_average[1h]) > 90")
		assertPrometheusRule(t, client, "redis-memory-usage-will-max-in-4-hours", ruleNs, "RedisMemoryUsageMaxIn4Hours", "critical", alertFor30Mins,
			"(predict_linear(sum by (instanceID) (cro_redis_memory_usage_percentage_average{job='operator-metrics-service'})[2h:1m], 6 * 3600) >= 100) and on (instanceID) (cro_redis_memory_usage_percentage_average{job='operator-metrics-service'} > 85)")
	})

	t.Run("skips when useClusterStorage is true", func(t *testing.T) {
		clusterInst := inst.DeepCopy()
		clusterInst.Spec.UseClusterStorage = "true"
		client := utils.NewTestClient(scheme)
		if err := createRedisMemoryUsageAlerts(context.TODO(), client, clusterInst, redis, log); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rule := &monv1.PrometheusRule{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: "redis-memory-usage-high", Namespace: ruleNs}, rule)
		if !k8serr.IsNotFound(err) {
			t.Fatalf("expected no rules when useClusterStorage is true, got err=%v", err)
		}
	})

	t.Run("returns error when configmap get fails", func(t *testing.T) {
		mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
		mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
			return fmt.Errorf("generic error")
		}
		if err := createRedisMemoryUsageAlerts(context.TODO(), mockClient, inst, redis, log); err == nil {
			t.Fatal("expected error when config get fails")
		}
	})
}

func assertPrometheusRule(t *testing.T, client k8sclient.Client, name, ns, alertName, severity, alertFor, expr string) {
	t.Helper()
	rule := &monv1.PrometheusRule{}
	if err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, rule); err != nil {
		t.Fatalf("failed to get prometheus rule %s: %v", name, err)
	}
	if len(rule.Spec.Groups) != 1 || len(rule.Spec.Groups[0].Rules) != 1 {
		t.Fatalf("rule %s: expected 1 group with 1 rule, got %+v", name, rule.Spec.Groups)
	}
	got := rule.Spec.Groups[0].Rules[0]
	if got.Alert != alertName {
		t.Errorf("rule %s alert = %s, want %s", name, got.Alert, alertName)
	}
	if got.Labels["severity"] != severity {
		t.Errorf("rule %s severity = %s, want %s", name, got.Labels["severity"], severity)
	}
	if got.For == nil || string(*got.For) != alertFor {
		t.Errorf("rule %s for = %v, want %s", name, got.For, alertFor)
	}
	if got.Expr != intstr.FromString(expr) {
		t.Errorf("rule %s expr = %s, want %s", name, got.Expr.String(), expr)
	}
}
