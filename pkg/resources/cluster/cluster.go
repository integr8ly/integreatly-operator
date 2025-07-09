package cluster

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const clusterVersionName = "version"

func ClusterVersionBefore49(ctx context.Context, serverClient k8sclient.Client, log l.Logger) (bool, error) {

	clusterVersion, err := GetClusterVersionCR(ctx, serverClient)
	if err != nil {
		return false, err
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

func GetClusterVersionCR(ctx context.Context, serverClient k8sclient.Client) (*configv1.ClusterVersion, error) {
	clusterVersionCR := &configv1.ClusterVersion{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: clusterVersionName}, clusterVersionCR)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version: %w", err)
	}
	return clusterVersionCR, nil
}

func getCurrentVersion(versionHistory []configv1.UpdateHistory, log l.Logger) (float64, error) {

	if len(versionHistory) < 1 {
		log.Error("Error getting cluster version, no version history", nil, nil)
		return 0, errors.New("Error getting cluster version, no version history")
	}
	s := strings.Split(versionHistory[0].Version, ".")
	if len(s) < 2 {
		log.Error("Error splitting cluster version history", l.Fields{"versionHistory": versionHistory[0].Version}, nil)
		return 0, errors.New("Error splitting cluster version history " + versionHistory[0].Version)
	}
	currentVersion := s[0] + "." + s[1]
	log.Infof("Current cluster ", l.Fields{"version": currentVersion})

	version, err := strconv.ParseFloat(currentVersion, 64)
	if err != nil {
		log.Error("Error parsing cluster version", l.Fields{"currentVersion": currentVersion}, err)
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
		return "", fmt.Errorf("no platform information found for type %s", infra.Status.PlatformStatus.Type)

	}
}

func GetPlatformType(ctx context.Context, c client.Client) (configv1.PlatformType, error) {
	infra, err := GetClusterInfrastructure(ctx, c)
	if err != nil {
		return "", err
	}
	return infra.Status.PlatformStatus.Type, nil
}

func GetExternalClusterId(cr *configv1.ClusterVersion) (configv1.ClusterID, error) {
	if cr.Spec.ClusterID != "" {
		return cr.Spec.ClusterID, nil
	}
	return "", fmt.Errorf("external cluster ID not found")
}

func GetClusterVersion(cr *configv1.ClusterVersion) (string, error) {
	if cr.Status.Desired.Version != "" {
		return cr.Status.Desired.Version, nil
	}
	return "", fmt.Errorf("desired.version not set in status block")
}
