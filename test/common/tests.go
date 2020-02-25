package common

var (
	ALL_TESTS = []TestCase{
		// Add all the tests that should be executed in both e2e and osd suites here.
		// It is an array so the tests will be executed in the same order as they defined here.
		{"Verify CRD Exists", TestIntegreatlyCRDExists},
		{"Verify alerts exist", TestIntegreatlyAlertsExist},
	}

	AFTER_INSTALL_TESTS = []TestCase{
		{"Verify Deployment resources have the expected replicas", TestDeploymentExpectedReplicas},
		{"Verify Deployment Config resources have the expected replicas", TestDeploymentConfigExpectedReplicas},
	}
)
