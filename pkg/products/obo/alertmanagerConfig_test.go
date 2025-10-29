package obo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/utils"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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
)

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
			Type:                 string(integreatlyv1alpha1.InstallationTypeManagedApi),
		},
	}
}

func basicInstallationWithAlertEmailAddress() *integreatlyv1alpha1.RHMI {
	installation := basicInstallation()
	installation.Spec.AlertFromAddress = mockAlertFromAddress
	installation.Spec.AlertingEmailAddress = mockCustomerAlertingEmailAddress
	installation.Spec.AlertingEmailAddresses.CSSRE = mockAlertingEmailAddress
	installation.Spec.AlertingEmailAddresses.BusinessUnit = mockBUAlertingEmailAddress
	installation.Namespace = defaultInstallationNamespace
	return installation
}

func TestReconciler_reconcileAlertManagerSecrets(t *testing.T) {
	scheme, err := utils.NewTestScheme()
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
			Name:      config.AlertManagerConfigSecretName,
			Namespace: config.GetOboNamespace(installation.Namespace),
		},
		Data: map[string][]byte{
			"alertmanager.yaml": []byte("global:\n  smtp_from: noreply-alert@devshift.org"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	alertmanagerService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerServiceName,
			Namespace: config.GetOboNamespace(installation.Namespace),
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
		"Subject":               `{{template "email.integreatly.subject" . }}`,
		"clusterID":             clusterID,
		"clusterName":           clusterName,
		"clusterConsole":        clusterConsoleRoute,
		"html":                  `{{ template "email.integreatly.html" . }}`,
	})

	templatePath := GetTemplatePath()
	path := fmt.Sprintf("%s/%s", templatePath, config.AlertManagerCustomTemplatePath)

	// generate alertmanager custom email template
	testEmailConfigContents, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed reading file: %v", err)
	}

	testEmailConfigContentsStr := string(testEmailConfigContents)
	cluster_vars := map[string]string{
		"${CLUSTER_NAME}":    clusterName,
		"${CLUSTER_ID}":      clusterID,
		"${CLUSTER_CONSOLE}": clusterConsoleRoute,
	}

	for name, val := range cluster_vars {
		testEmailConfigContentsStr = strings.ReplaceAll(testEmailConfigContentsStr, name, val)
	}

	testSecretData, err := templateUtil.LoadTemplate(config.AlertManagerConfigTemplatePath)
	if err != nil {
		t.Errorf("Failed loading template: %v", err)
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		setup        func() error
		want         integreatlyv1alpha1.StatusPhase
		wantFn       func(c k8sclient.Client) error
		wantErr      string
	}{
		{
			name: "succeeds when smtp secret cannot be found",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, pagerdutySecret, dmsSecret, alertmanagerService, clusterInfra, clusterVersion, clusterRoute)
			},
			wantErr: "",
			want:    integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "fails when pager duty secret cannot be found",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, alertmanagerService)
			},
			wantErr: "could not obtain pagerduty credentials secret: secrets \"test-pd\" not found",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when pager duty service key is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				return utils.NewTestClient(scheme, smtpSecret, emptyPagerdutySecret, alertmanagerService)
			},
			wantErr: "secret key is undefined in pager duty secret",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "succeeds when dead mans snitch secret cannot be found",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, alertmanagerService, clusterInfra, clusterVersion, clusterRoute)
			},
			wantErr: "",
			want:    integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "succeeds when dead mans snitch url is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, emptyDMSSecret, alertmanagerService, clusterInfra, clusterVersion, clusterRoute)
			},
			wantErr: "",
			want:    integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "awaiting components when alert manager route cannot be found",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, dmsSecret)
			},
			want: integreatlyv1alpha1.PhaseAwaitingComponents,
		},
		{
			name: "fails when alert manager service fails to be retrieved",
			serverClient: func() k8sclient.Client {
				return &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return fmt.Errorf("test")
					},
				}
			},
			wantErr: "failed to fetch alert manager service: test",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails cluster infra cannot  be retrieved",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerService)
			},
			want: integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "secret created successfully",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerService, clusterInfra, clusterVersion, clusterRoute)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: config.AlertManagerConfigSecretName, Namespace: config.GetOboNamespace(installation.Namespace)}, configSecret); err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[config.AlertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[config.AlertManagerConfigSecretFileName]), string(testSecretData))
				}
				if !bytes.Equal(configSecret.Data[config.AlertManagerEmailTemplateSecretFileName], []byte(testEmailConfigContentsStr)) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[config.AlertManagerEmailTemplateSecretFileName]), testEmailConfigContentsStr)
				}
				return nil
			},
		},
		{
			name: "secret data is overridden if already exists",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerService, alertmanagerConfigSecret, clusterInfra, clusterVersion, clusterRoute)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: config.AlertManagerConfigSecretName, Namespace: config.GetOboNamespace(installation.Namespace)}, configSecret); err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[config.AlertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[config.AlertManagerConfigSecretFileName]), string(testSecretData))
				}
				if !bytes.Equal(configSecret.Data[config.AlertManagerEmailTemplateSecretFileName], []byte(testEmailConfigContentsStr)) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[config.AlertManagerEmailTemplateSecretFileName]), testEmailConfigContentsStr)
				}
				return nil
			},
		},
		{
			name: "alert address env override is successful",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerService, clusterInfra, clusterVersion, clusterRoute)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: config.AlertManagerConfigSecretName, Namespace: config.GetOboNamespace(installation.Namespace)}, configSecret); err != nil {
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
					"Subject":               `{{template "email.integreatly.subject" . }}`,
					"clusterID":             clusterID,
					"clusterName":           clusterName,
					"clusterConsole":        clusterConsoleRoute,
					"html":                  `{{ template "email.integreatly.html" . }}`,
				})

				templatePath := GetTemplatePath()
				path := fmt.Sprintf("%s/%s", templatePath, config.AlertManagerCustomTemplatePath)

				// generate alertmanager custom email template
				testEmailConfigContents, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				testEmailConfigContentsStr := string(testEmailConfigContents)
				cluster_vars := map[string]string{
					"${CLUSTER_NAME}":    clusterName,
					"${CLUSTER_ID}":      clusterID,
					"${CLUSTER_CONSOLE}": clusterConsoleRoute,
				}

				for name, val := range cluster_vars {
					testEmailConfigContentsStr = strings.ReplaceAll(testEmailConfigContentsStr, name, val)
				}

				testSecretData, err := templateUtil.LoadTemplate(config.AlertManagerConfigTemplatePath)
				if err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[config.AlertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[config.AlertManagerConfigSecretFileName]), string(testSecretData))
				}
				if !bytes.Equal(configSecret.Data[config.AlertManagerEmailTemplateSecretFileName], []byte(testEmailConfigContentsStr)) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[config.AlertManagerEmailTemplateSecretFileName]), testEmailConfigContentsStr)
				}

				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			serverClient := tt.serverClient()

			got, err := ReconcileAlertManagerSecrets(context.TODO(), serverClient, installation)
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
	scheme, err := utils.NewTestScheme()
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
				return utils.NewTestClient(scheme)
			},
			wantErr: "could not obtain pagerduty credentials secret: secrets \"test-pd\" not found",
		},
		{
			name: "fails when pager duty service key is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				return utils.NewTestClient(scheme, emptyPagerdutySecret)
			},
			wantErr: "secret key is undefined in pager duty secret",
		},

		{
			name: "fails when pager duty service key - value is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				emptyPagerdutySecret.Data["serviceKey"] = []byte("")
				return utils.NewTestClient(scheme, emptyPagerdutySecret)
			},
			wantErr: "secret key is undefined in pager duty secret",
		},
		{
			name: "secret read successfully - from pager duty operator secret",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, pagerdutySecret)
			},
			want: "test",
		},
		{
			name: "secret read successfully - from cssre pager duty operator secret",
			serverClient: func() k8sclient.Client {
				cssrePagerDutySecret := pagerdutySecret.DeepCopy()
				cssrePagerDutySecret.Data = make(map[string][]byte, 0)
				cssrePagerDutySecret.Data["serviceKey"] = []byte("cssre-pg-secret")
				return utils.NewTestClient(scheme, cssrePagerDutySecret)
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
	scheme, err := utils.NewTestScheme()
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
				return utils.NewTestClient(scheme)
			},
			wantErr: "could not obtain dead mans snitch credentials secret: secrets \"test-dms\" not found",
		},
		{
			name: "fails when dead man switch secret url is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				return utils.NewTestClient(scheme, emptyDMSSecret)
			},
			wantErr: "url is undefined in dead mans snitch secret",
		},

		{
			name: "fails when dead man switch secret SNITCHH_URL - value is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				emptyDMSSecret.Data["SNITCH_URL"] = []byte("")
				return utils.NewTestClient(scheme, emptyDMSSecret)
			},
			wantErr: "url is undefined in dead mans snitch secret",
		},
		{
			name: "secret read successfully - from dead man switch operator secret",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, dmsSecret)
			},
			want: "https://example.com",
		},
		{
			name: "secret read successfully - from cssre dead man switch operator secret",
			serverClient: func() k8sclient.Client {
				cssreDMSSecret := dmsSecret.DeepCopy()
				cssreDMSSecret.Data = make(map[string][]byte, 0)
				cssreDMSSecret.Data["url"] = []byte("https://example-cssredms-secret.com")
				return utils.NewTestClient(scheme, cssreDMSSecret)
			},
			want: "https://example-cssredms-secret.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			serverClient := tt.serverClient()

			got, err := GetDMSSecret(context.TODO(), serverClient, *installation)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("GetDMSSecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDMSSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSmtpHost(t *testing.T) {
	installation := basicInstallation()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.example.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name       string
		smtpSecret func() *corev1.Secret
		want       string
	}{
		{
			name: "getSmtpHost returns default value when host value is an empty string",
			smtpSecret: func() *corev1.Secret {
				emptySmtpSecret := smtpSecret.DeepCopy()
				emptySmtpSecret.Data = map[string][]byte{}
				emptySmtpSecret.Data["host"] = []byte("")
				return emptySmtpSecret
			},
			want: "smtp.example.com",
		},
		{
			name: "getSmtpHost returns default value when SmtpSecret data map is nil",
			smtpSecret: func() *corev1.Secret {
				invalidSmtpSecret := smtpSecret.DeepCopy()
				invalidSmtpSecret.Data = nil
				return invalidSmtpSecret
			},
			want: "smtp.example.com",
		},
		{
			name: "getSmtpHost returns host value from SmtpSecret",
			smtpSecret: func() *corev1.Secret {
				correctSmtpSecret := smtpSecret.DeepCopy()
				correctSmtpSecret.Data = map[string][]byte{}
				correctSmtpSecret.Data["host"] = []byte("smtpTest")
				return correctSmtpSecret
			},
			want: "smtpTest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			smtpSecret := tt.smtpSecret()

			got := getSmtpHost(smtpSecret)
			if got != tt.want {
				t.Errorf("getSmtpHost() got = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_getSmtpPort(t *testing.T) {
	installation := basicInstallation()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.example.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name       string
		smtpSecret func() *corev1.Secret
		want       string
	}{
		{
			name: "getSmtpPort returns default value when port value is an empty string",
			smtpSecret: func() *corev1.Secret {
				emptySmtpSecret := smtpSecret.DeepCopy()
				emptySmtpSecret.Data = map[string][]byte{}
				emptySmtpSecret.Data["port"] = []byte("")
				return emptySmtpSecret
			},
			want: "587",
		},
		{
			name: "getSmtpHost returns default value when SmtpSecret data map is nil",
			smtpSecret: func() *corev1.Secret {
				invalidSmtpSecret := smtpSecret.DeepCopy()
				invalidSmtpSecret.Data = nil
				return invalidSmtpSecret
			},
			want: "587",
		},
		{
			name: "getSmtpPort returns port value from SmtpSecret",
			smtpSecret: func() *corev1.Secret {
				correctSmtpSecret := smtpSecret.DeepCopy()
				correctSmtpSecret.Data = map[string][]byte{}
				correctSmtpSecret.Data["port"] = []byte("420")
				return correctSmtpSecret
			},
			want: "420",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			smtpSecret := tt.smtpSecret()

			got := getSmtpPort(smtpSecret)
			if got != tt.want {
				t.Errorf("getSmtpPort() got = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_getSmtpUsername(t *testing.T) {
	installation := basicInstallation()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.example.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name       string
		smtpSecret func() *corev1.Secret
		want       string
	}{
		{
			name: "getSmtpUsername returns default value when username value is an empty string",
			smtpSecret: func() *corev1.Secret {
				emptySmtpSecret := smtpSecret.DeepCopy()
				emptySmtpSecret.Data = map[string][]byte{}
				emptySmtpSecret.Data["username"] = []byte("")
				return emptySmtpSecret
			},
			want: "smtp_username",
		},
		{
			name: "getSmtpUsername returns default value when SmtpSecret data map is nil",
			smtpSecret: func() *corev1.Secret {
				invalidSmtpSecret := smtpSecret.DeepCopy()
				invalidSmtpSecret.Data = nil
				return invalidSmtpSecret
			},
			want: "smtp_username",
		},
		{
			name: "getSmtpUsername returns username value from SmtpSecret",
			smtpSecret: func() *corev1.Secret {
				correctSmtpSecret := smtpSecret.DeepCopy()
				correctSmtpSecret.Data = map[string][]byte{}
				correctSmtpSecret.Data["username"] = []byte("smtpTestUsername")
				return correctSmtpSecret
			},
			want: "smtpTestUsername",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			smtpSecret := tt.smtpSecret()

			got := getSmtpUsername(smtpSecret)
			if got != tt.want {
				t.Errorf("getSmtpUsername() got = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_getSmtpPassword(t *testing.T) {
	installation := basicInstallation()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.example.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name       string
		smtpSecret func() *corev1.Secret
		want       string
	}{
		{
			name: "getSmtpPassword returns default value when password value is an empty string",
			smtpSecret: func() *corev1.Secret {
				emptySmtpSecret := smtpSecret.DeepCopy()
				emptySmtpSecret.Data = map[string][]byte{}
				emptySmtpSecret.Data["password"] = []byte("")
				return emptySmtpSecret
			},
			want: "smtp_password",
		},
		{
			name: "getSmtpPassword returns default value when SmtpSecret data map is nil",
			smtpSecret: func() *corev1.Secret {
				invalidSmtpSecret := smtpSecret.DeepCopy()
				invalidSmtpSecret.Data = nil
				return invalidSmtpSecret
			},
			want: "smtp_password",
		},
		{
			name: "getSmtpPassword returns password value from SmtpSecret",
			smtpSecret: func() *corev1.Secret {
				correctSmtpSecret := smtpSecret.DeepCopy()
				correctSmtpSecret.Data = map[string][]byte{}
				correctSmtpSecret.Data["password"] = []byte("smtpTestPassword")
				return correctSmtpSecret
			},
			want: "smtpTestPassword",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			smtpSecret := tt.smtpSecret()

			got := getSmtpPassword(smtpSecret)
			if got != tt.want {
				t.Errorf("getSmtpPassword got = %v, want %v", got, tt.want)
			}
		})
	}

}
