package functional

import "github.com/integr8ly/integreatly-operator/test/common"

var (
	FUNCTIONAL_TESTS = []common.TestCase{
		{Description: "F01 - Verify AWS rds resources exist and are in expected state", Test: AWSRDSResourcesExistTest},
		{Description: "F03 - Verify AWS elasticache resources exist and are in expected state", Test: AWSElasticacheResourcesExistTest},
		{Description: "A21 - Verify AWS maintenance and backup windows", Test: CROStrategyOverrideAWSResourceTest},
		{Description: "A25 - Verify standalone RHMI VPC exists and is configured properly", Test: TestStandaloneVPCExists},
		{Description: "N06 - Verify Legacy cluster VPC is configured properly", Test: TestLegacyClusterVPC},
		{Description: "F04 - Verify AWS s3 blob storage resources exist", Test: TestAWSs3BlobStorageResourcesExist},
	}
	MULTIAZ_TESTS = []common.TestCase{
		{Description: "F09 - Verify correct pod distribution on Multi-AZ cluster", Test: TestMultiAZPodDistribution},
	}
)
