package common

import (
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

// All tests are be linked[1] to the integreatly-test-cases[2] repo by using the same ID
// 1. https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases#how-to-automate-a-test-case-and-link-it-back
// 2. https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases
var (
	ALL_TESTS = []TestSuite{
		{
			[]TestCase{
				// Add all tests that can be executed prior to a completed installation here
				{"Verify RHMI CRD Exists", TestIntegreatlyCRDExists},
				{"Verify RHMI Config CRD Exists", TestRHMIConfigCRDExists},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged, v1alpha1.InstallationTypeManagedApi},
		},
	}

	HAPPY_PATH_TESTS = []TestSuite{
		//Add all happy path tests to be executed after RHMI installation is completed here
		{
			[]TestCase{
				{"F06 - Verify Replicas Scale correctly in Apicurito", TestReplicasInApicurito},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged},
		},
		{
			[]TestCase{
				{"E09 - Verify customer dashboards exist", TestIntegreatlyCustomerDashboardsExist},
				/*FLAKY on RHOAM*/ {"E10 - Verify Customer Grafana Route is accessible", TestCustomerGrafanaExternalRouteAccessible},
				{"A32 - Validate SSO config", TestSSOconfig},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
		{
			[]TestCase{
				// Keep test as first on the list, as it ensures that all products are reported as complete
				{"A01 - Verify that all stages in the integreatly-operator CR report completed", TestIntegreatlyStagesStatus},
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
				{"A18 - Verify RHMI Config CRs Successful", TestRHMIConfigCRs},
				{"A22 - Verify RHMI Config Updates CRO Strategy Override Config Map", TestRHMIConfigCROStrategyOverride},
				{"A26 - Verify Sendgrid Credentials Are Configured Properly", TestSendgridCredentialsAreValid},
				/*FLAKY on RHMI*/ {"C01 - Verify Alerts are not pending or firing apart from DeadMansSwitch", TestIntegreatlyAlertsPendingOrFiring},
				{"C04 - Verify Alerts exist", TestIntegreatlyAlertsExist},
				{"E01 - Verify Middleware Grafana Route is accessible", TestGrafanaExternalRouteAccessible},
				{"E02 - Verify that all dashboards are installed and all the graphs are filled with data", TestDashboardsData},
				{"E03 - Verify middleware dashboards exist", TestIntegreatlyMiddelewareDashboardsExist},
				/*FLAKY on RHMI/RHOAM*/ {"E05 - Verify Grafana Route returns dashboards", TestGrafanaExternalRouteDashboardExist},
				{"F02 - Verify PodDisruptionBudgets exist", TestIntegreatlyPodDisruptionBudgetsExist},
				{"Verify servicemonitors are cloned in monitoring namespace and rolebindings are created", TestServiceMonitorsCloneAndRolebindingsExist},
				/*FLAKY on RHMI*/ {"Verify Alerts are not firing during or after installation apart from DeadMansSwitch", TestIntegreatlyAlertsFiring},
				{"Verify prometheus metrics scrapped", TestMetricsScrappedByPrometheus},
				{"A27 + A28 - Verify pod priority class is created and set", TestPriorityClass},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged, v1alpha1.InstallationTypeManagedApi},
		},
	}

	IDP_BASED_TESTS = []TestSuite{
		{
			[]TestCase{
				{"A16 - Custom first broker login flow", TestAuthDelayFirstBrokerLogin},
				{"B03 - Verify RHMI Developer User Permissions are Correct", TestRHMIDeveloperUserPermissions},
				{"B04 - Verify Dedicated Admin User Permissions are Correct", TestDedicatedAdminUserPermissions},
				{"B06 - Verify users with no email get default email", TestDefaultUserEmail},
				{"H03 - Verify 3scale CRUDL permissions", Test3ScaleCrudlPermissions},
				{"H07 - ThreeScale User Promotion", Test3ScaleUserPromotion},
				{"Verify Network Policy allows cross NS access to SVC", TestNetworkPolicyAccessNSToSVC},
				{"H11 - Verify 3scale SMTP config", Test3ScaleSMTPConfig},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged, v1alpha1.InstallationTypeManagedApi},
		},
		{
			[]TestCase{
				{"B05 - Verify Codeready CRUDL permissions", TestCodereadyCrudlPermisssions},
				{"H05 - Verify Fuse CRUDL permissions", TestFuseCrudlPermissions},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged},
		},
	}

	SCALABILITY_TESTS = []TestSuite{
		{
			[]TestCase{
				{"F05 - Verify Replicas Scale correctly in Threescale", TestReplicasInThreescale},
				{"F08 - Verify Replicas Scale correctly in RHSSO", TestReplicasInRHSSO},
				{"F08 - Verify Replicas Scale correctly in User SSO", TestReplicasInUserSSO},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManaged, v1alpha1.InstallationTypeManagedApi},
		},

		{
			[]TestCase{
				{"A34 - Verify QUOTA values", TestQuotaValues},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
	}

	FAILURE_TESTS = []TestCase{
		{"C03 - Verify that alerting mechanism works", TestIntegreatlyAlertsMechanism},
	}

	DESTRUCTIVE_TESTS = []TestCase{
		// Add all destructive tests here that should not be executed as part of the happy path tests
		{"J03 - Verify namespaces restored when deleted", TestNamespaceRestoration},
	}
)
