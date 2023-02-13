package functional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/test/common"
	configv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	networkResourceType = "_network"
)

type strategyMap struct {
	CreateStrategy json.RawMessage `json:"createStrategy"`
}

func getExpectedPostgres(installType string, installationName string) []string {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		// expected postgres resources provisioned per product
		return []string{
			fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
		}
	} else {
		// expected postgres resources provisioned per product
		return []string{
			fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, installationName),
		}
	}
}

func getExpectedRedis(installType string, installationName string) []string {
	// expected redis resources provisioned per product
	commonRedis := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, installationName),
	}
	return commonRedis
}

/*
Each resource provisioned contains an annotation with the resource ID
This function iterates over a list of expected resource CR's
Returns a list of resource ID's, these ID's can be used when testing AWS or GCP resources
*/
func GetRedisInstanceData(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) (map[string]string, []error) {
	var foundErrors []error
	var instanceData map[string]string
	for _, redisName := range getExpectedRedis(rhmi.Spec.Type, rhmi.Name) {
		redis := &crov1.Redis{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: redisName}, redis); err != nil {
			foundErrors = append(foundErrors, fmt.Errorf("failed to find %s redis cr: %w", redisName, err))
		}
		if redis.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Errorf("found redis %q with phase %q but expected %q; message: %s", redisName, redis.Status.Phase, croTypes.PhaseComplete, redis.Status.Message))
		}
		resourceID, err := getCROAnnotation(redis)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Errorf("redis cr %q does not contain a resource id annotation: %w", redisName, err))
		}
		instanceData[resourceID] = redis.Status.Version
	}
	return instanceData, foundErrors
}

/*
Each resource provisioned contains an annotation with the resource ID
This function iterates over a list of expected resource CR's
Returns a list of resource ID's, these ID's can be used when testing postgres resources
*/
func GetPostgresInstanceData(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) (map[string]string, []error) {
	var foundErrors []error
	var instanceData map[string]string
	for _, pgName := range getExpectedPostgres(rhmi.Spec.Type, rhmi.Name) {
		postgres := &crov1.Postgres{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: pgName}, postgres); err != nil {
			foundErrors = append(foundErrors, fmt.Errorf("failed to find %s postgres cr: %w", pgName, err))
		}
		if postgres.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Errorf("found postgres %q with phase %q but expected %q; message: %s", pgName, postgres.Status.Phase, croTypes.PhaseComplete, postgres.Status.Message))
		}
		resourceID, err := getCROAnnotation(postgres)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Errorf("postgres cr %q does not contain a resource id annotation: %w", pgName, err))
		}
		instanceData[resourceID] = postgres.Status.Version
	}
	return instanceData, foundErrors
}

// return resource identifier annotation from cr
func getCROAnnotation(instance metav1.Object) (string, error) {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		return "", errors.New(fmt.Sprintf("annotations for %s can not be nil", instance.GetName()))
	}

	for k, v := range annotations {
		if "resourceIdentifier" == k {
			return v, nil
		}
	}
	return "", errors.New(fmt.Sprintf("no resource identifier found for resource %s", instance.GetName()))
}

func getStrategyForResource(configMap *v1.ConfigMap, resourceType, tier string) (*strategyMap, error) {
	rawStrategyMapping := configMap.Data[resourceType]
	if rawStrategyMapping == "" {
		return nil, fmt.Errorf("strategy for resource type: %s is not defined", resourceType)
	}
	var strategyMapping map[string]*strategyMap
	if err := json.Unmarshal([]byte(rawStrategyMapping), &strategyMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal strategy mapping for resource type %s: %v", resourceType, err)
	}
	if strategyMapping[tier] == nil {
		return nil, fmt.Errorf("no strategy found for deployment type: %s and deployment tier: %s", resourceType, tier)
	}
	return strategyMapping[tier], nil
}

// GetClustersAvailableZones returns a map containing zone names that are currently available
func GetClustersAvailableZones(nodes *v1.NodeList) map[string]bool {
	zones := make(map[string]bool)
	for _, node := range nodes.Items {
		if isNodeWorkerAndReady(node) {
			for labelName, labelValue := range node.Labels {
				if labelName == "topology.kubernetes.io/zone" {
					zones[labelValue] = true
				}
			}
		}
	}
	return zones
}

func verifyCidrBlockIsInAllowedRange(cidrBlock string, allowedCidrRanges []string) error {

	_, cidr, err := net.ParseCIDR(cidrBlock)
	if err != nil {
		return fmt.Errorf("error parsing cidr %s", cidrBlock)
	}

	for _, cidrRanges := range allowedCidrRanges {
		_, cidrRangeNet, err := net.ParseCIDR(cidrRanges)
		if err != nil {
			return fmt.Errorf("error parsing cidr %s", cidrBlock)
		}
		if cidrRangeNet.Contains(cidr.IP) {
			return nil
		}
	}
	return fmt.Errorf("%s is not in the expected cidr range", cidrBlock)
}

func checkForOverlappingCidrBlocks(vpcCidrBlock, clusterCIDRBlock string) error {
	_, vpcCidr, err := net.ParseCIDR(vpcCidrBlock)
	if err != nil {
		return fmt.Errorf("error parsing vpc cidr block: %s", vpcCidr)
	}

	_, clusterCidr, err := net.ParseCIDR(clusterCIDRBlock)
	if err != nil {
		return fmt.Errorf("error parsing cluster cidr block: %s", clusterCidr)
	}

	if vpcCidr.Contains(clusterCidr.IP) || clusterCidr.Contains(vpcCidr.IP) {
		return fmt.Errorf("vpc cidr block (%s) overlaps with the cluster cidr block: (%s)", vpcCidr, clusterCidr)
	}

	return nil
}

func getClusterID(ctx context.Context, client client.Client) (string, error) {
	infra := &configv1.Infrastructure{}
	err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		return "", fmt.Errorf("failed to get aws region: %w", err)
	}
	return infra.Status.InfrastructureName, nil
}

func getExpectedBackupBucketResourceName(installationName string) string {
	return fmt.Sprintf("backups-blobstorage-%s", installationName)
}

func getExpectedThreeScaleBucketResourceName(installationName string) string {
	return fmt.Sprintf("threescale-blobstorage-%s", installationName)
}

// Shared functions for AWS s3 and GCP cloud storage

func getExpectedBlobStorage(installType string, installationName string) []string {

	// 3scale blob storage
	threescaleBlobStorage := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, installationName),
	}

	return threescaleBlobStorage
}

// GetCloudObjectStorageBlobStorageResourceIDs - used to get Blob Storage Resource IDs (buckets)
// for cloud object storages: AWS S3 or GCP Cloud Storage
func GetCloudObjectStorageBlobStorageResourceIDs(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) ([]string, []string) {
	var foundErrors []string
	var foundResourceIDs []string

	expectedBlobStorage := getExpectedBlobStorage(rhmi.Spec.Type, rhmi.Name)

	for _, r := range expectedBlobStorage {
		// get blobStorage CR name
		blobStorage := &crov1.BlobStorage{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: r}, blobStorage); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfailed to find %s blobStorage cr : %v", r, err))
		}
		// ensure phase is completed
		if blobStorage.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfound %s blobStorage not ready with phase: %s, message: %s", r, blobStorage.Status.Phase, blobStorage.Status.Message))
		}
		// return resource id
		resourceID, err := getCROAnnotation(blobStorage)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\n%s blobStorage cr does not contain a resource id annotation: %v", r, err))
		}
		// populate the array
		foundResourceIDs = append(foundResourceIDs, resourceID)
	}

	return foundResourceIDs, foundErrors
}
