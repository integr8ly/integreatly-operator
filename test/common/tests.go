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
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
	}

	HAPPY_PATH_TESTS = []TestSuite{
		//Add all happy path tests to be executed after RHMI installation is completed here
		{
			[]TestCase{
				{"A32 - Validate SSO config", TestSSOconfig},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
		{
			[]TestCase{
				{"E10 - Verify Customer Grafana Route is accessible", TestCustomerGrafanaExternalRouteAccessible},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
		{
			[]TestCase{
				// Keep test as first on the list, as it ensures that all products are reported as complete
				{"A01 - Verify that all stages in the integreatly-operator CR report completed", TestIntegreatlyStagesStatus},
				{"A03 - Verify all namespaces have been created with the correct name", TestNamespaceCreated},
				{"A05 - Verify product operator version", TestProductOperatorVersions},
				{"A07 - Verify product versions", TestProductVersions},
				{"A08 - Verify all products routes are created", TestIntegreatlyRoutesExist},
				{"A09 - Verify Subscription Install Plan Strategy", TestSubscriptionInstallPlanType},
				{"A10 - Verify CRO Postgres CRs Successful", TestCROPostgresSuccessfulState},
				{"A11 - Verify CRO Redis CRs Successful", TestCRORedisSuccessfulState},
				{"A13 - Verify Deployment resources have the expected replicas", TestDeploymentExpectedReplicas},
				{"A15 - Verify Stateful Set resources have the expected replicas", TestStatefulSetsExpectedReplicas},
				{"A26 - Verify Sendgrid Credentials Are Configured Properly", TestSendgridCredentialsAreValid},
				{"C01 - Verify Alerts are not pending or firing apart from DeadMansSwitch", TestIntegreatlyAlertsPendingOrFiring},
				{"F02 - Verify PodDisruptionBudgets exist", TestIntegreatlyPodDisruptionBudgetsExist},
				{"A27 + A28 - Verify pod priority class is created and set", TestPriorityClass},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
		{
			[]TestCase{
				{"M01 - Verify multitenancy works as expected", TestMultitenancy},
				//{"MT02 - Performance test simulate parallel Tenants creation", TestMultitenancyPerformance},
				// MT02 test will be used for manual Performance verification Only. Not include in Test suite!
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeMultitenantManagedApi},
		},
		{
			[]TestCase{
				{"Validate resource requirements are set", ValidateResourceRequirements},
				{"Verify addon instance status conditions", TestStatusConditions},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
	}

	OBSERVABILITY_TESTS = []TestSuite{
		//Observability related tests are kept separate from HAPPY_PATH because we skip these tests when IN_PROW is true.
		//This is because the prow checks run on OCP clusters and OCP clusters don't install OBO by default. See MGDAPI-5783.
		{
			[]TestCase{
				{"Test if cluster package is available", TestClusterPackageAvailable},
				{"Test RHMI installation CR metric", TestRHMICRMetrics},
				{"A06 - Verify PVC", TestPVClaims},
				{"C04 - Verify Alerts exist", TestIntegreatlyAlertsExist},
				{"C10B - Verify Prometheus blackbox targets", TestAdditionalBlackboxTargets},
				{"C08B - Verify alert links to SOPs", TestSOPUrls},
				{"Verify Alerts are not firing during or after installation apart from DeadMansSwitch", TestIntegreatlyAlertsFiring},
				{"Verify prometheus metrics scrapped", TestMetricsScrappedByPrometheus},
				{"E09 - Verify customer dashboards exist", TestIntegreatlyCustomerDashboardsExist},
				{"Verify ClusterObjectTemplates ready state", TestClusterObjectTemplateState},
				{"Verify package operator resource stability", TestPackageOperatorResourceStability},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
		{
			[]TestCase{
				{"M02B - Verify RHOAM version metric is exposed in Prometheus", TestRhoamVersionMetricExposed},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
	}

	//Threescale cluster scoped test suite to be used when Threescale becomes cluster scoped.
	THREESCALE_CLUSTER_SCOPED_TESTS = []TestSuite{
		{
			[]TestCase{
				{"H29 - Verify that backend can be created via backend CR", Test3scaleBackendViaCR},
				{"H30 - Verify that product can be created via product CR", Test3scaleProductViaCR},
				{"H31 - Verify that tenant can be created via tenant CR", Test3scaleTenantViaCr},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
	}

	IDP_BASED_TESTS = []TestSuite{
		{
			[]TestCase{
				{"B06 - Verify users with no email get default email", TestDefaultUserEmail},
				{"Verify Network Policy allows cross NS access to SVC", TestNetworkPolicyAccessNSToSVC},
				{"C19 - Validate creation of invalid username triggers alert", TestInvalidUserNameAlert},
				{"H34 - Verify 3scale custom SMTP full config", Test3ScaleCustomSMTPFullConfig},
				{"H35 - Verify 3scale custom SMTP partial config", Test3ScaleCustomSMTPPartialConfig},
				{"H24 - Verify selfmanaged Apicast", TestSelfmanagedApicast},
				{"A33 - Verify console links", TestConsoleLinks},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
		{
			[]TestCase{
				{"B01B - Verify users can login to products", TestProductLogins},
				{"B08B - Verify users can create a Realm in User SSO", TestUsersCreateRealmSSO},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
		{
			[]TestCase{
				{"A16 - Custom first broker login flow", TestAuthDelayFirstBrokerLogin},
				{"B03 - Verify RHMI Developer User Permissions are Correct", TestRHMIDeveloperUserPermissions},
				{"B04 - Verify Dedicated Admin User Permissions are Correct", TestDedicatedAdminUserPermissions},
				{"B09 - Verify dedicated admin users are synced with User SSO", TestDedicatedAdminUsersSyncedSSO},
				{"H03 - Verify 3scale CRUDL permissions", Test3ScaleCrudlPermissions},
				{"H07 - ThreeScale User Promotion", Test3ScaleUserPromotion},
				// Keep H11 as last 3scale IDP Test as test causes 3scale deployments to be rescaled at the end of test
				// Can potentially cause subsequent tests be flaky due to waiting for 3scale deployments to complete
				{"H11 - Verify 3scale SMTP config", Test3ScaleSMTPConfig},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
	}

	SCALABILITY_TESTS = []TestSuite{
		{
			[]TestCase{
				{"F05 - Verify Replicas Scale correctly in Threescale", TestReplicasInThreescale},
				{"F08 - Verify Replicas Scale correctly in RHSSO", TestReplicasInRHSSO},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
		{
			[]TestCase{
				{"F08 - Verify Replicas Scale correctly in User SSO", TestReplicasInUserSSO},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi},
		},
		{
			[]TestCase{
				{"A34 - Verify Quota values", TestQuotaValues},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
	}

	FAILURE_TESTS = []TestCase{
		{"C03 - Verify that alerting mechanism works", TestIntegreatlyAlertsMechanism},
	}

	DESTRUCTIVE_TESTS = []TestCase{
		// Add all destructive tests here that should not be executed as part of the happy path tests
		{"J03 - Verify namespaces restored when deleted", TestNamespaceRestoration},
		{"C14B - Verify 3scale UIBBT alerts", TestThreeScaleUIBBTAlerts},
	}

	AWS_SPECIFIC_TESTS = []TestSuite{
		{
			[]TestCase{
				{"A12 - Verify CRO BlobStorage CRs Successful", TestCROBlobStorageSuccessfulState},
			},
			[]v1alpha1.InstallationType{v1alpha1.InstallationTypeManagedApi, v1alpha1.InstallationTypeMultitenantManagedApi},
		},
	}
)
