package functional

import "github.com/integr8ly/integreatly-operator/test/common"

var (
	FUNCTIONAL_TESTS = []common.TestCase{
		{Description: "Verify AWS rds resources exist and are in expected state", Test: AWSRDSResourcesExistTest},
		{Description: "Verify AWS elasticache resources exist and are in expected state", Test: AWSElasticacheResourcesExistTest},
	}
)
