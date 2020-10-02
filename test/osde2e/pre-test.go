package osde2e

import "github.com/integr8ly/integreatly-operator/test/common"

var (
	OSD_E2E_PRE_TESTS = []common.TestCase{
		{Description: "Managed-API pre-test", Test: PreTest},
	}
)
