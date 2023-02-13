package functional

//some functions below were taken from CRO
import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/test/common"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultNumberOfExpectedSubnets = 2
	defaultIpRangeCIDRMask         = 22
	managedLabelKey                = "red-hat-managed"
	managedLabelValue              = "true"
	gcpCredsSecretName             = "cloud-resource-gcp-credentials"
)

func validateSubnetsCidrRangeAndOverlapWithStartegyMapCidr(strategyMapCIDR *net.IPNet, subnets []*computepb.Subnetwork) error {
	for i := range subnets {
		_, subnetCIDR, err := net.ParseCIDR(croResources.SafeStringDereference(subnets[i].IpCidrRange))
		if err != nil {
			return fmt.Errorf("failed to parse cluster subnet into cidr")
		}
		if !isValidCIDRRange(subnetCIDR) {
			return fmt.Errorf("subnet cidr %s is out of range, block sizes must be `/22` or lower, subnet: %s", subnetCIDR.String(), *subnets[i].Name)
		}
		if subnetCIDR.Contains(strategyMapCIDR.IP) || strategyMapCIDR.Contains(subnetCIDR.IP) {
			return fmt.Errorf("strategy map cidr %s overlaps with subnet cidr %s , subnet: %s", strategyMapCIDR.String(), subnetCIDR.String(), *subnets[i].Name)
		}
		fmt.Printf("Subnet verified: %s", *subnets[i].Name)
	}
	return nil
}

func isValidCIDRRange(validateCIDR *net.IPNet) bool {
	mask, _ := validateCIDR.Mask.Size()
	return mask <= defaultIpRangeCIDRMask
}

// parses a subnet URL in the format:
// https://www.googleapis.com/compute/v1/projects/my-project-1234/regions/my-region/subnetworks/my-subnet-name
func parseSubnetUrl(subnetUrl string) (string, string, error) {
	parsed, err := url.Parse(subnetUrl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse subnet url %s , %w", subnetUrl, err)
	}
	var name, region string
	path := strings.Split(parsed.Path, "/")
	for i := range path {
		if path[i] == "regions" {
			region = path[i+1]
		}
		if path[i] == "subnetworks" {
			name = path[i+1]
			break
		}
	}
	if name == "" || region == "" {
		return "", "", fmt.Errorf("failed to retrieve subnetwork name from URL")
	}
	return name, region, nil
}

func getGCPCredentials(ctx context.Context, client client.Client) ([]byte, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: gcpCredsSecretName, Namespace: common.RHOAMOperatorNamespace}, secret); err != nil {
		return nil, fmt.Errorf("failed getting secret %s from ns %s: %w", gcpCredsSecretName, common.RHOAMOperatorNamespace, err)
	}
	serviceAccountJson := secret.Data["service_account.json"]
	if len(serviceAccountJson) == 0 {
		return nil, errors.New("gcp credentials secret can't be empty")
	}
	return serviceAccountJson, nil
}

func labelsContain(labels map[string]string, key, value string) bool {
	for k, v := range labels {
		if k == key && v == value {
			return true
		}
	}
	return false
}
