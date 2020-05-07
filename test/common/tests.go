package common

// All tests are be linked[1] to the integreatly-test-cases[2] repo by using the same ID
// 1. https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases#how-to-automate-a-test-case-and-link-it-back
// 2. https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases
var (
	ALL_TESTS = []TestCase{
		// Add all tests that can be executed prior to a completed installation here
		{"Verify RHMI CRD Exists", TestIntegreatlyCRDExists},
		{"Verify RHMI Config CRD Exists", TestRHMIConfigCRDExists},
	}

	HAPPY_PATH_TESTS = []TestCase{
		// Add all happy path tests to be executed after RHMI installation is completed here
		{"A18 - Verify RHMI Config CRs Successful", TestRHMIConfigCRs},
		{"B03 - Verify RHMI Developer User Permissions are Correct", TestRHMIDeveloperUserPermissions},
		{"B04 - Verify Dedicated Admin User Permissions are Correct", TestDedicatedAdminUserPermissions},
		{"A13 - Verify Deployment resources have the expected replicas", TestDeploymentExpectedReplicas},
		{"A14 - Verify Deployment Config resources have the expected replicas", TestDeploymentConfigExpectedReplicas},
		{"A15 - Verify Stateful Set resources have the expected replicas", TestStatefulSetsExpectedReplicas},
		{"E03 - Verify dashboards exist", TestIntegreatlyDashboardsExist},
		{"A10 - Verify CRO Postgres CRs Successful", TestCROPostgresSuccessfulState},
		{"A11 - Verify CRO Redis CRs Successful", TestCRORedisSuccessfulState},
		{"A12 - Verify CRO BlobStorage CRs Successful", TestCROBlobStorageSuccessfulState},
		{"F02 - Verify PodDisruptionBudgets exist", TestIntegreatlyPodDisruptionBudgetsExist},
		{"A08 - Verify all products routes are created", TestIntegreatlyRoutesExist},
		{"E01 - Verify Grafana Route is accessible", TestGrafanaExternalRouteAccessible},
		{"E05 - Verify Grafana Route returns dashboards", TestGrafanaExternalRouteDashboardExist},
		{"C04 - Verify Alerts exist", TestIntegreatlyAlertsExist},
		{"C01 - Verify Alerts are not firing apart from DeadMansSwitch", TestIntegreatlyAlertsFiring},
		{"B06 - Verify users with no email get default email", TestDefaultUserEmail},
		{"H05 - Verify Fuse CRUDL permissions", TestFuseCrudlPermissions},
		{"B05 - Verify Codeready CRUDL permissions", TestCodereadyCrudlPermisssions},
		{"A09 - Verify Subscription Install Plan Strategy", TestSubscriptionInstallPlanType},
		{"Verify Network Policy allows cross NS access to SVC", TestNetworkPolicyAccessNSToSVC},
		{"A16 - Custom first broker login flow", TestAuthDelayFirstBrokerLogin},
		{"F05 - Verify Replicas Scale correctly in Threescale", TestReplicasInThreescale},
		{"F06 - Verify Replicas Scale correctly in Apicurito", TestReplicasInApicurito},
		{"F08 - Verify Replicas Scale correctly in RHSSO and user SSO", TestReplicasInRHSSOAndUserSSO},
		{"A06 - Verify PVC", TestPVClaims},
	}

	DESTRUCTIVE_TESTS = []TestCase{
		// Add all destructive tests here that should not be executed as part of the happy path tests
		{"J03 - Verify namespaces restored when deleted", TestNamespaceRestoration},
	}
)
