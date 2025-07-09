package marin3r

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	totalRequestsMetric = "authorized_calls"
)

func (r *Reconciler) newAlertsReconciler(grafanaDashboardURL string, namespace string) (resources.AlertReconciler, error) {

	requestsAllowedPerSecond, err := r.getRateLimitInSeconds(r.RateLimitConfig.Unit, r.RateLimitConfig.RequestsPerUnit)
	if err != nil {
		return nil, err
	}

	alerts, err := mapAlertsConfiguration(r.log, namespace, r.RateLimitConfig.Unit, r.RateLimitConfig.RequestsPerUnit, requestsAllowedPerSecond, r.AlertsConfig, grafanaDashboardURL, r.installation.Spec.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create alerts from configuration: %w", err)
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "3Scale",
		Installation: r.installation,
		Log:          r.log,
		Alerts:       alerts,
	}, nil
}

// mapAlertsConfiguration maps each value from alertsConfig into a
// resources.AlertConfiguration object, resulting into a list of the
// prometheus alerts to be created
func mapAlertsConfiguration(logger l.Logger, namespace, rateLimitUnit string, rateLimitRequestsPerUnit uint32, requestsAllowedPerSecond float64, alertsConfig map[string]*marin3rconfig.AlertConfig, grafanaDashboardURL string, installationName string) ([]resources.AlertConfiguration, error) {
	result := make([]resources.AlertConfiguration, 0, len(alertsConfig))

	for alertName, alertConfig := range alertsConfig {

		if installationName == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
			alertName = "marin3r-" + alertName
		}

		switch alertConfig.Type {
		case marin3rconfig.AlertTypeSpike:
			expr := fmt.Sprintf(
				"max_over_time((sum(increase(authorized_calls[1m])) + sum(increase(limited_calls[1m])))[%s:]) > %d",
				alertConfig.Period, rateLimitRequestsPerUnit)
			annotations := map[string]string{
				"message":        fmt.Sprintf("hard limit of %d breached at least once in the last %s", rateLimitRequestsPerUnit, alertConfig.Period),
				"grafanaConsole": grafanaDashboardURL,
			}
			alert := mapSpikeAlert(alertConfig, alertName, namespace, expr, annotations, installationName)
			result = append(result, alert)
		case marin3rconfig.AlertTypeThreshold:

			usageFrequencyMins, err := intervalToMinutes(alertConfig.Period)
			if err != nil {
				return nil, err
			}
			requestsAllowedOverTimePeriod := requestsAllowedPerSecond * float64(usageFrequencyMins*60)

			minRateValue, maxRateValue, err := parsePercentageRange(
				alertConfig.Threshold.MinRate,
				alertConfig.Threshold.MaxRate,
			)
			if err != nil {
				return nil, err
			}

			lowerExpr := increaseExpr(totalRequestsMetric, alertConfig.Period, ">=", requestsAllowedOverTimePeriod, &minRateValue)
			upperExpr := increaseExpr(totalRequestsMetric, alertConfig.Period, "<=", requestsAllowedOverTimePeriod, maxRateValue)

			// Get the complete expression by ANDing the lower and the upper if the
			// upper limit is set, if not, assign the lower one
			expr := *lowerExpr
			upperMessage := "100%"
			if upperExpr != nil {
				expr = fmt.Sprintf("%s and %s", expr, *upperExpr)
				upperMessage = *alertConfig.Threshold.MaxRate
			}
			annotations := map[string]string{
				"message": fmt.Sprintf(
					"Total API usage in your API Management service is between %s and %s of the allowable threshold, %d requests per %s, during the last %s",
					alertConfig.Threshold.MinRate, upperMessage, rateLimitRequestsPerUnit, rateLimitUnit, alertConfig.Period,
				),
				"grafanaConsole": grafanaDashboardURL,
			}
			alert := mapThresholdAlert(alertConfig, alertName, namespace, expr, annotations, installationName)

			result = append(result, alert)
		default:
			logger.Infof("Unsupported Alert Type found", l.Fields{"alertName": alertName})
		}

	}
	return result, nil
}

func mapSpikeAlert(alertConfig *marin3rconfig.AlertConfig, alertName string, namespace string, expr string, annotations map[string]string, installationName string) resources.AlertConfiguration {
	return resources.AlertConfiguration{
		AlertName: alertName,
		GroupName: "ratelimit-spike.rules",
		Namespace: namespace,
		Interval:  alertConfig.Period,
		Rules: []monv1.Rule{
			{
				Alert:       alertConfig.RuleName,
				Annotations: annotations,
				Expr:        intstr.FromString(expr),
				Labels:      map[string]string{"severity": alertConfig.Level, "product": installationName},
			},
		},
	}
}

func mapThresholdAlert(alertConfig *marin3rconfig.AlertConfig, alertName string, namespace string, expr string, annotations map[string]string, installationName string) resources.AlertConfiguration {
	return resources.AlertConfiguration{
		AlertName: alertName,
		GroupName: "api-usage.rules",
		Namespace: namespace,
		Rules: []monv1.Rule{
			{
				Alert:       alertConfig.RuleName,
				Annotations: annotations,
				Expr:        intstr.FromString(expr),
				Labels:      map[string]string{"severity": alertConfig.Level, "product": installationName},
			},
		},
	}
}

func increaseExpr(totalRequestsMetric, period string, comparisonOperator string, requestsAllowedOverTimePeriod float64, percenteageLimit *int) *string {
	if percenteageLimit == nil {
		return nil
	}

	result := fmt.Sprintf(
		"(sum(increase(%s[%s])) %s (%f / 100 * %d))",
		totalRequestsMetric, period, comparisonOperator, requestsAllowedOverTimePeriod, *percenteageLimit,
	)

	return &result
}

func (r *Reconciler) getRateLimitInSeconds(rateLimitUnit string, rateLimitRequestsPerUnit uint32) (float64, error) {
	if rateLimitUnit == "second" {
		return float64(rateLimitRequestsPerUnit), nil
	} else if rateLimitUnit == "minute" {
		return float64(rateLimitRequestsPerUnit) / 60, nil
	} else if rateLimitUnit == "hour" {
		return float64(rateLimitRequestsPerUnit) / (60 * 60), nil
	} else if rateLimitUnit == "day" {
		return float64(rateLimitRequestsPerUnit) / (60 * 60 * 24), nil
	} else {
		err := fmt.Errorf("unexpected Rate Limit Unit %s, while creating 3scale api usage alerts", rateLimitUnit)
		r.log.Error("", nil, err)
		return 0, err
	}
}

// intervalToMinutes parses an interval string made up from a number and a unit
// that can be "m" for minutes, or "h" for hours, and returns the value in minutes
// or an error if the string representation is invalid
func intervalToMinutes(interval string) (uint32, error) {
	re := regexp.MustCompile(`(?m)([0-9]+)([a-zA-Z])$`)
	matches := re.FindAllStringSubmatch(interval, -1)

	if len(matches) == 0 || len(matches[0]) != 3 {
		return 0, fmt.Errorf("invalid value for interval %s", interval)
	}

	intervalValueStr := matches[0][1]
	intervalUnit := matches[0][2]

	var multiplier int
	switch strings.ToLower(intervalUnit) {
	case "m":
		multiplier = 1
	case "h":
		multiplier = 60
	default:
		return 0, fmt.Errorf("invalid value for interval unit %s, must be m or h", intervalUnit)
	}

	intervalValue, err := strconv.Atoi(intervalValueStr)
	if err != nil {
		return 0, err
	}
	result := int64(intervalValue) * int64(multiplier)
	if result < 0 || result > math.MaxUint32 {
		return 0, fmt.Errorf("calculated interval value %d is out of the valid range for uint32", result)
	}
	return uint32(result), nil
}

// parsePercentage parses and validates a percentage string by extracting
// the numeric value and validating that it's in a correct value for a percentage
func parsePercentage(percentage *string) (*int, error) {
	if percentage == nil {
		return nil, nil
	}

	var re = regexp.MustCompile(`(?m)([0-9]+)%$`)
	matches := re.FindAllStringSubmatch(*percentage, -1)

	if len(matches) == 0 || len(matches[0]) != 2 {
		return nil, fmt.Errorf("invalid value for percentage %s", *percentage)
	}

	result, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return nil, nil
	}

	if result < 0 || result > 100 {
		return nil, fmt.Errorf("%d is an invalid percentage", result)
	}

	return &result, nil
}

// parsePercentageRange parses both min and max as percentages, and validates
// that min is less than or equal to max
func parsePercentageRange(min string, max *string) (int, *int, error) {
	minValue, err := parsePercentage(&min)
	if err != nil {
		return 0, nil, err
	}

	maxValue, err := parsePercentage(max)
	if err != nil {
		return 0, nil, err
	}

	if maxValue != nil && *minValue > *maxValue {
		return 0, nil, fmt.Errorf("min value %d must be less than or equal to max value %d", minValue, maxValue)
	}

	return *minValue, maxValue, nil
}
