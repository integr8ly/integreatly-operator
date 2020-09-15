package common

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

// All tests are be linked[1] to the integreatly-test-cases[2] repo by using the same ID
// 1. https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases#how-to-automate-a-test-case-and-link-it-back
// 2. https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases
var (
	ALL_TESTS = []TestCase{
		// Add all tests that can be executed prior to a completed installation here
		{"Verify RHMI CRD Exists", TestIntegreatlyCRDExists},
		{"Verify RHMI Config CRD Exists", TestRHMIConfigCRDExists},
	}

	HAPPY_PATH_TESTS = []TestSuite{
		//Add all happy path tests to be executed after RHMI installation is completed here
		{
			[]TestCase{
				{"B05 - Verify Codeready CRUDL permissions", TestCodereadyCrudlPermisssions},
				{"F06 - Verify Replicas Scale correctly in Apicurito", TestReplicasInApicurito},
				{"H05 - Verify Fuse CRUDL permissions", TestFuseCrudlPermissions},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged},
		},
		{
			[]TestCase{
				{"A01 - Verify that all stages in the integreatly-operator CR report completed", TestIntegreatlyStagesStatus}, // Keep test as first on the list, as it ensures that all products are reported as complete
				{"Test RHMI installation CR metric", TestRHMICRMetrics},
				{"A03 - Verify all namespaces have been created with the correct name", TestNamespaceCreated},
				{"A05 - Verify product operator version", TestProductOperatorVersions},
				{"A06 - Verify PVC", TestPVClaims},
				{"A07 - Verify product versions", TestProductVersions},
				{"A08 - Verify all products routes are created", TestIntegreatlyRoutesExist},
				{"A09 - Verify Subscription Install Plan Strategy", TestSubscriptionInstallPlanType},
				{"A10 - Verify CRO Postgres CRs Successful", TestCROPostgresSuccessfulState},
				{"A11 - Verify CRO Redis CRs Successful", TestCRORedisSuccessfulState},
				{"A12 - Verify CRO BlobStorage CRs Successful", TestCROBlobStorageSuccessfulState},
				{"A13 - Verify Deployment resources have the expected replicas", TestDeploymentExpectedReplicas},
				{"A14 - Verify Deployment Config resources have the expected replicas", TestDeploymentConfigExpectedReplicas},
				{"A15 - Verify Stateful Set resources have the expected replicas", TestStatefulSetsExpectedReplicas},
				{"A16 - Custom first broker login flow", TestAuthDelayFirstBrokerLogin},
				{"A18 - Verify RHMI Config CRs Successful", TestRHMIConfigCRs},
				{"A22 - Verify RHMI Config Updates CRO Strategy Override Config Map", TestRHMIConfigCROStrategyOverride},
				{"A26 - Verify Sendgrid Credentials Are Configured Properly", TestSendgridCredentialsAreValid},
				{"B03 - Verify RHMI Developer User Permissions are Correct", TestRHMIDeveloperUserPermissions},
				{"B04 - Verify Dedicated Admin User Permissions are Correct", TestDedicatedAdminUserPermissions},
				{"B06 - Verify users with no email get default email", TestDefaultUserEmail},
				{"C01 - Verify Alerts are not pending or firing apart from DeadMansSwitch", TestIntegreatlyAlertsPendingOrFiring},
				{"C03 - Verify that alerting mechanism works", TestIntegreatlyAlertsMechanism},
				{"C04 - Verify Alerts exist", TestIntegreatlyAlertsExist},
				{"E01 - Verify Grafana Route is accessible", TestGrafanaExternalRouteAccessible},
				{"E02 - Verify that all dashboards are installed and all the graphs are filled with data", TestDashboardsData},
				{"E03 - Verify dashboards exist", TestIntegreatlyDashboardsExist},
				{"E05 - Verify Grafana Route returns dashboards", TestGrafanaExternalRouteDashboardExist},
				{"F02 - Verify PodDisruptionBudgets exist", TestIntegreatlyPodDisruptionBudgetsExist},
				{"F05 - Verify Replicas Scale correctly in Threescale", TestReplicasInThreescale},
				{"F08 - Verify Replicas Scale correctly in RHSSO and user SSO", TestReplicasInRHSSOAndUserSSO},
				{"H03 - Verify 3scale CRUDL permissions", Test3ScaleCrudlPermissions},
				{"H07 - ThreeScale User Promotion", Test3ScaleUserPromotion},
				{"H11 - Verify 3scale SMTP config", Test3ScaleSMTPConfig},
				{"Verify servicemonitors are cloned in monitoring namespace and rolebindings are created", TestServiceMonitorsCloneAndRolebindingsExist},
				{"Verify Alerts are not firing during or after installation apart from DeadMansSwitch", TestIntegreatlyAlertsFiring},
				{"Verify Network Policy allows cross NS access to SVC", TestNetworkPolicyAccessNSToSVC},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged, v1alpha1.InstallationTypeManagedApi},
		},
	}

	DESTRUCTIVE_TESTS = []TestCase{
		// Add all destructive tests here that should not be executed as part of the happy path tests
		{"J03 - Verify namespaces restored when deleted", TestNamespaceRestoration},
	}
)
