package resources

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

// GetForcedReconcileTimeOrDefault returns envar for reconcile time else returns default time
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

// GetMetricReconcileTimeOrDefault returns envar for reconcile time else returns default time
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

// BuildInfraName builds and returns an id used for infra resources
func BuildInfraName(ctx context.Context, c client.Client, postfix string, n int) (string, error) {
	// get cluster id
	clusterID, err := GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "error getting clusterID")
	}
	return ShortenString(fmt.Sprintf("%s-%s", clusterID, postfix), n), nil
}

func BuildInfraNameFromObject(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterID, err := GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve cluster identifier")
	}
	return ShortenString(fmt.Sprintf("%s-%s-%s", clusterID, om.Namespace, om.Name), n), nil
}

func BuildTimestampedInfraNameFromObject(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterID, err := GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve timestamped cluster identifier")
	}
	curTime := time.Now().Unix()
	return ShortenString(fmt.Sprintf("%s-%s-%s-%d", clusterID, om.Namespace, om.Name, curTime), n), nil
}

func BuildTimestampedInfraNameFromObjectCreation(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterID, err := GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve timestamped cluster identifier")
	}
	return ShortenString(fmt.Sprintf("%s-%s-%s-%s", clusterID, om.Namespace, om.Name, om.GetObjectMeta().GetCreationTimestamp()), n), nil
}
