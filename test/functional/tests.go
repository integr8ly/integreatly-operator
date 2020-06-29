package functional

import "github.com/integr8ly/integreatly-operator/test/common"

var (
	FUNCTIONAL_TESTS = []common.TestCase{
		{Description: "F01 - Verify AWS rds resources exist and are in expected state", Test: AWSRDSResourcesExistTest},
		{Description: "F03 - Verify AWS elasticache resources exist and are in expected state", Test: AWSElasticacheResourcesExistTest},
		{Description: "A21 - Verify AWS maintenance and backup windows", Test: CROStrategyOverrideAWSResourceTest},
		{Description: "Verify AWS Standalone VPC exists", Test: TestStandaloneVPCExists},
		//TODO add F04 - Verify AWS s3 resources exist and are in expected state
	}
)
