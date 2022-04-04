package resources

import (
	"context"
	"errors"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

const clusterVersionName = "version"

func ClusterVersionBefore49(ctx context.Context, serverClient k8sclient.Client, log l.Logger) (bool, error) {

	clusterVersion := &configv1.ClusterVersion{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: clusterVersionName}, clusterVersion); err != nil {
		return false, fmt.Errorf("failed to fetch version: %w", err)
	}

	currentVersion, err := getCurrentVersion(clusterVersion.Status.History, log)
	if err != nil {
		return false, err
	}

	if currentVersion >= 4.9 {
		return false, nil
	}
	return true, nil
}

func getCurrentVersion(versionHistory []configv1.UpdateHistory, log l.Logger) (float64, error) {

	if len(versionHistory) < 1 {
		log.Error("Error getting cluster version, no version history", nil)
		return 0, errors.New("Error getting cluster version, no version history")
	}
	s := strings.Split(versionHistory[0].Version, ".")
	if len(s) < 2 {
		log.Errorf("Error splitting cluster version history", l.Fields{"versionHistory": versionHistory[0].Version}, nil)
		return 0, errors.New("Error splitting cluster version history " + versionHistory[0].Version)
	}
	currentVersion := s[0] + "." + s[1]
	log.Infof("Current cluster ", l.Fields{"version": currentVersion})

	version, err := strconv.ParseFloat(currentVersion, 64)
	if err != nil {
		log.Errorf("Error parsing cluster version", l.Fields{"currentVersion": currentVersion}, err)
		return 0, errors.New("Error parsing cluster version " + currentVersion)
	}

	return version, nil
}

func GetClusterInfrastructure(ctx context.Context, c client.Client) (*configv1.Infrastructure, error) {
	infra := &configv1.Infrastructure{}
	if err := c.Get(ctx, types.NamespacedName{Name: "cluster"}, infra); err != nil {
		return nil, fmt.Errorf("failed to retrieve cluster infrastructure: %w", err)
	}
	return infra, nil
}

func GetClusterType(infra *configv1.Infrastructure) (string, error) {

	switch infra.Status.PlatformStatus.Type {
	case configv1.AWSPlatformType:
		for _, tag := range infra.Status.PlatformStatus.AWS.ResourceTags {
			if tag.Key == "red-hat-clustertype" {
				return tag.Value, nil
			}
		}
		return "", fmt.Errorf("key \"red-hat-clustertype\" not in AWS resource tags")
	default:
		return "Unknown", fmt.Errorf("no platform information found for type %s", infra.Status.PlatformStatus.Type)

	}
}
