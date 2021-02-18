package resources

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	errorUtil "github.com/pkg/errors"
)

const (
	EnvForceReconcileTimeout   = "ENV_FORCE_RECONCILE_TIMEOUT"
	EnvMetricsReconcileTimeout = "ENV_METRIC_RECONCILE_TIMEOUT"
	DefaultTagKeyPrefix        = "integreatly.org/"
	// Set the reconcile duration for this controller.
	// Currently it will be called once every 5 minutes
	MetricsWatchDuration = 5 * time.Minute
)

//GetForcedReconcileTimeOrDefault returns envar for reconcile time else returns default time
func GetForcedReconcileTimeOrDefault(defaultTo time.Duration) time.Duration {
	recTime, exist := os.LookupEnv(EnvForceReconcileTimeout)
	if exist {
		rt, err := strconv.ParseInt(recTime, 10, 64)
		if err != nil {
			return defaultTo
		}
		return time.Duration(rt)
	}
	return defaultTo
}

//GetMetricReconcileTimeOrDefault returns envar for reconcile time else returns default time
func GetMetricReconcileTimeOrDefault(defaultTo time.Duration) time.Duration {
	recTime, exist := os.LookupEnv(EnvMetricsReconcileTimeout)
	if exist {
		rt, err := strconv.ParseInt(recTime, 10, 64)
		if err != nil {
			return defaultTo
		}
		return time.Duration(rt)
	}
	return defaultTo
}

func GeneratePassword() (string, error) {
	generatedPassword, err := uuid.NewRandom()
	if err != nil {
		return "", errorUtil.Wrap(err, "error generating password")
	}
	return strings.Replace(generatedPassword.String(), "-", "", 10), nil
}

func GetOrganizationTag() string {
	// get the environment from the CR
	organizationTag, exists := os.LookupEnv("TAG_KEY_PREFIX")
	if !exists {
		organizationTag = DefaultTagKeyPrefix
	}
	return organizationTag
}
