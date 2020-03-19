package functional

import "github.com/integr8ly/integreatly-operator/test/common"

var (
	FUNCTIONAL_TESTS = []common.TestCase{
		{"Verify AWS Resources exist and are in expected state", AWSResourcesExistTest},
	}
)
