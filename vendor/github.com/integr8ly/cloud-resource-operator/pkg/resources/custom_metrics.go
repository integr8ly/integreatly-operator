package resources

import (
	"time"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	errorUtil "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	sleepytime = 3600
)

var (
	//MetricVecs create the map of vectors
	MetricVecs map[string]prometheus.GaugeVec
	logger     *logrus.Entry
)

func init() {
	StartGaugeVector()
}

//StartGaugeVector periodic loop that is wiping all known vectors.
func StartGaugeVector() {
	MetricVecs = map[string]prometheus.GaugeVec{}
	logger = logrus.WithFields(logrus.Fields{"custom_metrics": "StartGaugeVector"})

	go func() {
		for {
			logger.Info("calling reset on all prometheus gauge vectors")
			for _, val := range MetricVecs {
				val.Reset()
			}
			time.Sleep(time.Duration(sleepytime) * time.Second)
		}
	}()
}

//SetMetric Set exports a Prometheus Gauge
func SetMetric(name string, labels map[string]string, value float64) error {
	// set vector value
	gv, ok := MetricVecs[name]
	if ok {
		gv.With(labels).Set(value)
		return nil
	}

	// create label array for vector creation
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	// the vector does not exist, create it, register and then add this gauge metric to the gauge vector
	gv = *prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, keys)
	customMetrics.Registry.MustRegister(gv)
	MetricVecs[name] = gv

	return nil
}

//SetMetricCurrentTime Set current time wraps set metric
func SetMetricCurrentTime(name string, labels map[string]string) error {
	if err := SetMetric(name, labels, float64(time.Now().UnixNano())/1e9); err != nil {
		return errorUtil.Wrap(err, "unable to set current time gauge vector")
	}
	return nil
}

func createPrometheusRuleObject(ruleName string, namespace string, groups []prometheusv1.RuleGroup) *prometheusv1.PrometheusRule {
	return &prometheusv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: namespace,
			Labels: map[string]string{
				"monitoring-key": "middleware",
			},
		},
		Spec: prometheusv1.PrometheusRuleSpec{
			Groups: groups,
		},
	}
}

// CreatePrometheusRule will create a PrometheusRule object
func CreatePrometheusRule(ruleName string, namespace string, alertRuleName string, description string, alertExp intstr.IntOrString, labels map[string]string) (*prometheusv1.PrometheusRule, error) {
	alertGroupName := alertRuleName + "Group"

	groups := []prometheusv1.RuleGroup{
		{
			Name: alertGroupName,
			Rules: []prometheusv1.Rule{
				{
					Alert:  alertRuleName,
					Expr:   alertExp,
					Labels: labels,
					Annotations: map[string]string{
						"description": description,
					},
				},
			},
		},
	}

	return createPrometheusRuleObject(ruleName, namespace, groups), nil
}
