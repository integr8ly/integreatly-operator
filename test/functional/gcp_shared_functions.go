package functional

//some functions below were taken from CRO
import (
	"context"
	"fmt"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/test/common"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"k8s.io/apimachinery/pkg/types"
	"net"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	defaultNumberOfExpectedSubnets = 2
	defaultIpRangeCIDRMask         = 22
	managedLabelKey                = "red-hat-managed"
	managedLabelValue              = "true"
)

func GetPostgresSqlInstancesIDsListFromCR(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) ([]string, []string) {
	var foundErrors []string
	var foundResourceIDs []string

	expectedPSql := getExpectedPSql(rhmi.Spec.Type, rhmi.Name)

	for _, r := range expectedPSql {
		// get pSql cr
		postgres := &crov1.Postgres{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: r}, postgres); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfailed to find %s postgres cr : %v", r, err))
		}
		// ensure phase is completed
		if postgres.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfound %s postgres not ready with phase: %s, message: %s", r, postgres.Status.Phase, postgres.Status.Message))
		}
		// return resource id
		resourceID, err := getCROAnnotation(postgres)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\n%s postgres cr does not contain a resource id annotation: %v", r, err))
		}
		// populate the array
		foundResourceIDs = append(foundResourceIDs, resourceID)
	}
	return foundResourceIDs, foundErrors
}

func getExpectedPSql(installType string, installationName string) []string {
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
