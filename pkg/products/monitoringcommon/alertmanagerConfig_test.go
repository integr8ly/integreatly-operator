package monitoringcommon

import (
	"bytes"
	"context"
	"fmt"
	observability "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	configv1 "github.com/openshift/api/config/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	v1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

const (
	mockSMTPSecretName               = "test-smtp"
	mockPagerdutySecretName          = "test-pd"
	mockDMSSecretName                = "test-dms"
	mockCustomerAlertingEmailAddress = "noreply-customer-test@rhmi-redhat.com"
	mockAlertingEmailAddress         = "noreply-test@rhmi-redhat.com"
	mockBUAlertingEmailAddress       = "noreply-bu-test@rhmi-redhat.com"
	mockAlertFromAddress             = "noreply-alert@devshift.org"

	defaultInstallationNamespace = "mock-namespace"
	alertManagerRouteName        = "mock-routename"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := observability.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := operatorsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := coreosv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := prometheusmonitoringv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rbac.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := configv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := routev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

func basicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			SMTPSecret:           mockSMTPSecretName,
			PagerDutySecret:      mockPagerdutySecretName,
			DeadMansSnitchSecret: mockDMSSecretName,
			Type:                 string(integreatlyv1alpha1.InstallationTypeManaged),
		},
	}
}

func basicInstallationWithAlertEmailAddress() *integreatlyv1alpha1.RHMI {
	installation := basicInstallation()
	installation.Spec.AlertFromAddress = mockAlertFromAddress
	installation.Spec.AlertingEmailAddress = mockCustomerAlertingEmailAddress
	installation.Spec.AlertingEmailAddresses.CSSRE = mockAlertingEmailAddress
	installation.Spec.AlertingEmailAddresses.BusinessUnit = mockBUAlertingEmailAddress
	return installation
}

func TestReconciler_reconcileAlertManagerSecrets(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := basicInstallationWithAlertEmailAddress()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.sendgrid.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockPagerdutySecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"serviceKey": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDMSSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"url": []byte("https://example.com"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	alertmanagerConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerConfigSecretName,
			Namespace: defaultInstallationNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}
	alertmanagerRoute := &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerRouteName,
			Namespace: defaultInstallationNamespace,
		},
		Spec: v1.RouteSpec{
			Host: "example.com",
		},
	}

	clusterInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterInfraName,
		},
		Status: configv1.InfrastructureStatus{
			InfrastructureName: "cluster-infra",
		},
	}

	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterVersionName,
		},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: "cluster-id",
		},
	}

	clusterRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      openShiftConsoleRoute,
			Namespace: openShiftConsoleNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: "example-console.com",
		},
	}

	clusterConsoleRoute := fmt.Sprintf(`https://%v`, clusterRoute.Spec.Host)
	clusterName := clusterInfra.Status.InfrastructureName
	clusterID := string(clusterVersion.Spec.ClusterID)

	templateUtil := NewTemplateHelper(map[string]string{
		"SMTPHost":              string(smtpSecret.Data["host"]),
		"SMTPPort":              string(smtpSecret.Data["port"]),
		"SMTPFrom":              mockAlertFromAddress,
		"SMTPUsername":          string(smtpSecret.Data["username"]),
		"SMTPPassword":          string(smtpSecret.Data["password"]),
		"PagerDutyServiceKey":   string(pagerdutySecret.Data["serviceKey"]),
		"DeadMansSnitchURL":     string(dmsSecret.Data["url"]),
		"SMTPToCustomerAddress": mockCustomerAlertingEmailAddress,
		"SMTPToSREAddress":      mockAlertingEmailAddress,
		"SMTPToBUAddress":       mockBUAlertingEmailAddress,
		"Subject":               fmt.Sprintf(`{{template "email.integreatly.subject" . }}`),
		"clusterID":             clusterID,
		"clusterName":           clusterName,
		"clusterConsole":        clusterConsoleRoute,
		"html":                  fmt.Sprintf(`{{ template "email.integreatly.html" . }}`),
	})

	templatePath := GetTemplatePath()
	path := fmt.Sprintf("%s/%s", templatePath, alertManagerCustomTemplatePath)

	// generate alertmanager custom email template
	testEmailConfigContents, err := ioutil.ReadFile(path)

	testEmailConfigContentsStr := string(testEmailConfigContents)
	cluster_vars := map[string]string{
		"${CLUSTER_NAME}":    clusterName,
		"${CLUSTER_ID}":      clusterID,
		"${CLUSTER_CONSOLE}": clusterConsoleRoute,
	}

	for name, val := range cluster_vars {
		testEmailConfigContentsStr = strings.ReplaceAll(testEmailConfigContentsStr, name, val)
	}

	testSecretData, err := templateUtil.LoadTemplate(alertManagerConfigTemplatePath)

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		setup        func() error
		want         integreatlyv1alpha1.StatusPhase
		wantFn       func(c k8sclient.Client) error
		wantErr      string
	}{
		{
			name: "fails when pager duty secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, alertmanagerRoute)
			},
			wantErr: "could not obtain pagerduty credentials secret: secrets \"test-pd\" not found",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when pager duty service key is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, emptyPagerdutySecret, alertmanagerRoute)
			},
			wantErr: "secret key is undefined in pager duty secret",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when dead mans snitch secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, alertmanagerRoute)
			},
			wantErr: "could not obtain dead mans snitch credentials secret: secrets \"test-dms\" not found",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when dead mans snitch url is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, emptyDMSSecret, alertmanagerRoute)
			},
			wantErr: "url is undefined in dead mans snitch secret",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "awaiting components when alert manager route cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret)
			},
			want: integreatlyv1alpha1.PhaseAwaitingComponents,
		},
		{
			name: "fails when alert manager route fails to be retrieved",
			serverClient: func() k8sclient.Client {
				return &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
						return fmt.Errorf("test")
					},
				}
			},
			wantErr: "could not obtain alert manager route: test",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails cluster infra cannot  be retrieved",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute)
			},
			want: integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "secret created successfully",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute, clusterInfra, clusterVersion, clusterRoute)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: alertManagerConfigSecretName, Namespace: defaultInstallationNamespace}, configSecret); err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[alertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerConfigSecretFileName]), string(testSecretData))
				}
				if !bytes.Equal(configSecret.Data[alertManagerEmailTemplateSecretFileName], []byte(testEmailConfigContentsStr)) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerEmailTemplateSecretFileName]), testEmailConfigContentsStr)
				}
				return nil
			},
		},
		{
			name: "secret data is overridden if already exists",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute, alertmanagerConfigSecret, clusterInfra, clusterVersion, clusterRoute)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: alertManagerConfigSecretName, Namespace: defaultInstallationNamespace}, configSecret); err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[alertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerConfigSecretFileName]), string(testSecretData))
				}
				if !bytes.Equal(configSecret.Data[alertManagerEmailTemplateSecretFileName], []byte(testEmailConfigContentsStr)) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerEmailTemplateSecretFileName]), testEmailConfigContentsStr)
				}
				return nil
			},
		},
		{
			name: "alert address env override is successful",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute, clusterInfra, clusterVersion, clusterRoute)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: alertManagerConfigSecretName, Namespace: defaultInstallationNamespace}, configSecret); err != nil {
					return err
				}

				clusterConsoleRoute := fmt.Sprintf(`https://%v`, clusterRoute.Spec.Host)
				clusterName := clusterInfra.Status.InfrastructureName
				clusterID := string(clusterVersion.Spec.ClusterID)

				templateUtil := NewTemplateHelper(map[string]string{
					"SMTPHost":              string(smtpSecret.Data["host"]),
					"SMTPPort":              string(smtpSecret.Data["port"]),
					"SMTPFrom":              mockAlertFromAddress,
					"SMTPUsername":          string(smtpSecret.Data["username"]),
					"SMTPPassword":          string(smtpSecret.Data["password"]),
					"PagerDutyServiceKey":   string(pagerdutySecret.Data["serviceKey"]),
					"DeadMansSnitchURL":     string(dmsSecret.Data["url"]),
					"SMTPToCustomerAddress": mockCustomerAlertingEmailAddress,
					"SMTPToSREAddress":      mockAlertingEmailAddress,
					"SMTPToBUAddress":       mockBUAlertingEmailAddress,
					"Subject":               fmt.Sprintf(`{{template "email.integreatly.subject" . }}`),
					"clusterID":             clusterID,
					"clusterName":           clusterName,
					"clusterConsole":        clusterConsoleRoute,
					"html":                  fmt.Sprintf(`{{ template "email.integreatly.html" . }}`),
				})

				templatePath := GetTemplatePath()
				path := fmt.Sprintf("%s/%s", templatePath, alertManagerCustomTemplatePath)

				// generate alertmanager custom email template
				testEmailConfigContents, err := ioutil.ReadFile(path)

				testEmailConfigContentsStr := string(testEmailConfigContents)
				cluster_vars := map[string]string{
					"${CLUSTER_NAME}":    clusterName,
					"${CLUSTER_ID}":      clusterID,
					"${CLUSTER_CONSOLE}": clusterConsoleRoute,
				}

				for name, val := range cluster_vars {
					testEmailConfigContentsStr = strings.ReplaceAll(testEmailConfigContentsStr, name, val)
				}

				testSecretData, err := templateUtil.LoadTemplate(alertManagerConfigTemplatePath)
				if err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[alertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerConfigSecretFileName]), string(testSecretData))
				}
				if !bytes.Equal(configSecret.Data[alertManagerEmailTemplateSecretFileName], []byte(testEmailConfigContentsStr)) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerEmailTemplateSecretFileName]), testEmailConfigContentsStr)
				}

				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			serverClient := tt.serverClient()

			got, err := ReconcileAlertManagerSecrets(context.TODO(), serverClient, installation, defaultInstallationNamespace, alertManagerRouteName)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("reconcileAlertManagerConfigSecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileAlertManagerConfigSecret() got = %v, want %v", got, tt.want)
			}
			if tt.wantFn != nil {
				if err := tt.wantFn(serverClient); err != nil {
					t.Errorf("reconcileAlertManagerConfigSecret() error = %v", err)
				}
			}
		})
	}
}

func TestReconciler_getPagerDutySecret(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := basicInstallation()

	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockPagerdutySecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"PAGERDUTY_KEY": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		setup        func() error
		want         string
		wantErr      string
	}{
		{
			name: "fails when pager duty secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme)
			},
			wantErr: "could not obtain pagerduty credentials secret: secrets \"test-pd\" not found",
		},
		{
			name: "fails when pager duty service key is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyPagerdutySecret)
			},
			wantErr: "secret key is undefined in pager duty secret",
		},

		{
			name: "fails when pager duty service key - value is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				emptyPagerdutySecret.Data["serviceKey"] = []byte("")
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyPagerdutySecret)
			},
			wantErr: "secret key is undefined in pager duty secret",
		},
		{
			name: "secret read successfully - from pager duty operator secret",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, pagerdutySecret)
			},
			want: "test",
		},
		{
			name: "secret read successfully - from cssre pager duty operator secret",
			serverClient: func() k8sclient.Client {
				cssrePagerDutySecret := pagerdutySecret.DeepCopy()
				cssrePagerDutySecret.Data = make(map[string][]byte, 0)
				cssrePagerDutySecret.Data["serviceKey"] = []byte("cssre-pg-secret")
				return fakeclient.NewFakeClientWithScheme(basicScheme, cssrePagerDutySecret)
			},
			want: "cssre-pg-secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			serverClient := tt.serverClient()

			got, err := getPagerDutySecret(context.TODO(), serverClient, *installation)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("getPagerDutySecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getPagerDutySecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_getDMSSecret(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := basicInstallation()

	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDMSSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"SNITCH_URL": []byte("https://example.com"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		setup        func() error
		want         string
		wantErr      string
	}{
		{
			name: "fails when dead man switch secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme)
			},
			wantErr: "could not obtain dead mans snitch credentials secret: secrets \"test-dms\" not found",
		},
		{
			name: "fails when dead man switch secret url is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyDMSSecret)
			},
			wantErr: "url is undefined in dead mans snitch secret",
		},

		{
			name: "fails when dead man switch secret SNITCHH_URL - value is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				emptyDMSSecret.Data["SNITCH_URL"] = []byte("")
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyDMSSecret)
			},
			wantErr: "url is undefined in dead mans snitch secret",
		},
		{
			name: "secret read successfully - from dead man switch operator secret",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, dmsSecret)
			},
			want: "https://example.com",
		},
		{
			name: "secret read successfully - from cssre dead man switch operator secret",
			serverClient: func() k8sclient.Client {
				cssreDMSSecret := dmsSecret.DeepCopy()
				cssreDMSSecret.Data = make(map[string][]byte, 0)
				cssreDMSSecret.Data["url"] = []byte("https://example-cssredms-secret.com")
				return fakeclient.NewFakeClientWithScheme(basicScheme, cssreDMSSecret)
			},
			want: "https://example-cssredms-secret.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			serverClient := tt.serverClient()

			got, err := getDMSSecret(context.TODO(), serverClient, *installation)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("getDMSSecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getDMSSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}
