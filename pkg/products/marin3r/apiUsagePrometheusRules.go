package marin3r

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	totalRequestsMetric = "ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits"
)

func (r *Reconciler) newAlertsReconciler() (resources.AlertReconciler, error) {

	requestsAllowedPerSecond, err := getRateLimitInSeconds(r.RateLimitConfig.Unit, r.RateLimitConfig.RequestsPerUnit)
	if err != nil {
		return nil, err
	}

	alerts, err := mapAlertsConfiguration(r.Config.GetNamespace(), r.RateLimitConfig.Unit, r.RateLimitConfig.RequestsPerUnit, requestsAllowedPerSecond, r.AlertsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create alerts from configuration: %w", err)
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "3Scale",
		Installation: r.installation,
		Logger:       r.logger,
		Alerts:       alerts,
	}, nil
}

// mapAlertsConfiguration maps each value from alertsConfig into a
// resources.AlertConfiguration object, resulting into a list of the
// prometheus alerts to be created
func mapAlertsConfiguration(namespace, rateLimitUnit string, rateLimitRequestsPerUnit, requestsAllowedPerSecond uint32, alertsConfig map[string]*marin3rconfig.AlertConfig) ([]resources.AlertConfiguration, error) {
	result := make([]resources.AlertConfiguration, 0, len(alertsConfig))

	for alertName, alertConfig := range alertsConfig {
		usageFrequencyMins, err := intervalToMinutes(alertConfig.Period)
		if err != nil {
			return nil, err
		}
		requestsAllowedOverTimePeriod := requestsAllowedPerSecond * usageFrequencyMins * 60

		minRateValue, maxRateValue, err := parsePercenteageRange(
			alertConfig.MinRate,
			alertConfig.MaxRate,
		)
		if err != nil {
			return nil, err
		}

		alert := resources.AlertConfiguration{
			AlertName: alertName,
			GroupName: "api-usage.rules",
			Interval:  alertConfig.Period,
			Namespace: namespace,
			Rules: []monitoringv1.Rule{
				{
					Alert: alertConfig.RuleName,
					Annotations: map[string]string{
						"message": fmt.Sprintf(
							"3Scale API usage is between %s and %s of the allowable threshold, %d requests per %s, during the last %s",
							alertConfig.MinRate, alertConfig.MaxRate, rateLimitRequestsPerUnit, rateLimitUnit, alertConfig.Period,
						),
					},
					Expr: intstr.FromString(fmt.Sprintf("(increase(%s[%s]) >= (%d / 100 * %d)) and (increase(%s[%s]) <=  (%d / 100 * %d))",
						totalRequestsMetric, alertConfig.Period, requestsAllowedOverTimePeriod, minRateValue, totalRequestsMetric, alertConfig.Period, requestsAllowedOverTimePeriod, maxRateValue,
					)),
					Labels: map[string]string{"severity": alertConfig.Level},
				},
			},
		}

		result = append(result, alert)
	}

	return result, nil
}

func getRateLimitInSeconds(rateLimitUnit string, rateLimitRequestsPerUnit uint32) (uint32, error) {
	if rateLimitUnit == "seconds" {
		return rateLimitRequestsPerUnit, nil
	} else if rateLimitUnit == "minute" {
		return rateLimitRequestsPerUnit * 60, nil
	} else if rateLimitUnit == "hour" {
		return rateLimitRequestsPerUnit * 60 * 60, nil
	} else if rateLimitUnit == "day" {
		return rateLimitRequestsPerUnit * 60 * 60 * 24, nil
	} else {
		logrus.Errorf("Unexpected Rate Limit Unit %v, while creating 3scale api usage alerts", rateLimitUnit)
		return 0, errors.New(fmt.Sprintf("Unexpected Rate Limit Unit %v, while creating 3scale api usage alerts", rateLimitUnit))
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
		break
	case "h":
		multiplier = 60
		break
	default:
		return 0, fmt.Errorf("invalid value for interval unit %s, must be m or h", intervalUnit)
	}

	intervalValue, err := strconv.Atoi(intervalValueStr)
	if err != nil {
		return 0, err
	}

	return uint32(intervalValue * multiplier), nil
}

// parsePercenteage parses and validates a percenteage string by extracting
// the numeric value and validating that it's in a correct value for a percenteage
func parsePercenteage(percenteage string) (int, error) {
	var re = regexp.MustCompile(`(?m)([0-9]+)%$`)
	matches := re.FindAllStringSubmatch(percenteage, -1)

	if len(matches) == 0 || len(matches[0]) != 2 {
		return 0, fmt.Errorf("invalid value for percenteage %s", percenteage)
	}

	result, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return 0, nil
	}

	if result < 0 || result > 100 {
		return 0, fmt.Errorf("%d is an invalid percenteage", result)
	}

	return result, nil
}

// parsePercenteageRange parses both min and max as percenteages, and validates
// that min is less than or equal to max
func parsePercenteageRange(min, max string) (int, int, error) {
	minValue, err := parsePercenteage(min)
	if err != nil {
		return 0, 0, err
	}

	maxValue, err := parsePercenteage(max)
	if err != nil {
		return 0, 0, err
	}

	if minValue > maxValue {
		return 0, 0, fmt.Errorf("min value %d must be less than or equal to max value %d", minValue, maxValue)
	}

	return minValue, maxValue, nil
}
