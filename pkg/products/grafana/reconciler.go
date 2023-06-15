package grafana

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	consolev1 "github.com/openshift/api/console/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/util/retry"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/version"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "customer-monitoring"
	defaultGrafanaName           = "grafana"
	defaultRoutename             = defaultGrafanaName + "-route"
	rateLimitDashBoardName       = "rate-limit"

	grafanaConsoleLink     = "grafana-user-console-link"
	grafanaIcon            = "data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI1LjIuMCwgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAzNyAzNyIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMzcgMzc7IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KCS5zdDB7ZmlsbDojRUUwMDAwO30KCS5zdDF7ZmlsbDojRkZGRkZGO30KPC9zdHlsZT4KPGc+Cgk8cGF0aCBkPSJNMjcuNSwwLjVoLTE4Yy00Ljk3LDAtOSw0LjAzLTksOXYxOGMwLDQuOTcsNC4wMyw5LDksOWgxOGM0Ljk3LDAsOS00LjAzLDktOXYtMThDMzYuNSw0LjUzLDMyLjQ3LDAuNSwyNy41LDAuNUwyNy41LDAuNXoiCgkJLz4KCTxnPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yNSwyMi4zN2MtMC45NSwwLTEuNzUsMC42My0yLjAyLDEuNWgtMS44NVYyMS41YzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYycy0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDIuNDhjMC4yNywwLjg3LDEuMDcsMS41LDIuMDIsMS41YzEuMTcsMCwyLjEyLTAuOTUsMi4xMi0yLjEyUzI2LjE3LDIyLjM3LDI1LDIyLjM3eiBNMjUsMjUuMzcKCQkJYy0wLjQ4LDAtMC44OC0wLjM5LTAuODgtMC44OHMwLjM5LTAuODgsMC44OC0wLjg4czAuODgsMC4zOSwwLjg4LDAuODhTMjUuNDgsMjUuMzcsMjUsMjUuMzd6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTIwLjUsMTYuMTJjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTIuMzhoMS45MWMwLjMyLDAuNzcsMS4wOCwxLjMxLDEuOTYsMS4zMQoJCQljMS4xNywwLDIuMTItMC45NSwyLjEyLTIuMTJzLTAuOTUtMi4xMi0yLjEyLTIuMTJjLTEuMDIsMC0xLjg4LDAuNzMtMi4wOCwxLjY5SDIwLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJQzE5Ljg3LDE1Ljg1LDIwLjE2LDE2LjEyLDIwLjUsMTYuMTJ6IE0yNSwxMS40M2MwLjQ4LDAsMC44OCwwLjM5LDAuODgsMC44OHMtMC4zOSwwLjg4LTAuODgsMC44OHMtMC44OC0wLjM5LTAuODgtMC44OAoJCQlTMjQuNTIsMTEuNDMsMjUsMTEuNDN6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTEyLjEyLDE5Ljk2di0wLjg0aDIuMzhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJzLTAuMjgtMC42Mi0wLjYyLTAuNjJoLTIuMzh2LTAuOTEKCQkJYzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYyaC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYzYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDNDMTEuODQsMjAuNTksMTIuMTIsMjAuMzEsMTIuMTIsMTkuOTYKCQkJeiBNMTAuODcsMTkuMzRIOS4xMnYtMS43NWgxLjc1VjE5LjM0eiIvPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yOC41LDE2LjM0aC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYwLjkxSDIyLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYyczAuMjgsMC42MiwwLjYyLDAuNjJoMi4zOAoJCQl2MC44NGMwLDAuMzUsMC4yOCwwLjYyLDAuNjIsMC42MmgzYzAuMzQsMCwwLjYyLTAuMjgsMC42Mi0wLjYydi0zQzI5LjEyLDE2LjYyLDI4Ljg0LDE2LjM0LDI4LjUsMTYuMzR6IE0yNy44NywxOS4zNGgtMS43NXYtMS43NQoJCQloMS43NVYxOS4zNHoiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwyMC44N2MtMC4zNCwwLTAuNjMsMC4yOC0wLjYzLDAuNjJ2Mi4zOGgtMS44NWMtMC4yNy0wLjg3LTEuMDctMS41LTIuMDItMS41CgkJCWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMmMwLjk1LDAsMS43NS0wLjYzLDIuMDItMS41aDIuNDhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTMKCQkJQzE3LjEyLDIxLjE1LDE2Ljg0LDIwLjg3LDE2LjUsMjAuODd6IE0xMiwyNS4zN2MtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4CgkJCVMxMi40OCwyNS4zNywxMiwyNS4zN3oiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwxMS44N2gtMi40MmMtMC4yLTAuOTctMS4wNi0xLjY5LTIuMDgtMS42OWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMgoJCQljMC44OCwwLDEuNjQtMC41NCwxLjk2LTEuMzFoMS45MXYyLjM4YzAsMC4zNSwwLjI4LDAuNjIsMC42MywwLjYyczAuNjItMC4yOCwwLjYyLTAuNjJ2LTNDMTcuMTIsMTIuMTUsMTYuODQsMTEuODcsMTYuNSwxMS44N3oKCQkJIE0xMiwxMy4xOGMtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4UzEyLjQ4LDEzLjE4LDEyLDEzLjE4eiIvPgoJPC9nPgoJPHBhdGggY2xhc3M9InN0MSIgZD0iTTE4LjUsMjIuNjJjLTIuMjcsMC00LjEzLTEuODUtNC4xMy00LjEyczEuODUtNC4xMiw0LjEzLTQuMTJzNC4xMiwxLjg1LDQuMTIsNC4xMlMyMC43NywyMi42MiwxOC41LDIyLjYyegoJCSBNMTguNSwxNS42MmMtMS41OCwwLTIuODgsMS4yOS0yLjg4LDIuODhzMS4yOSwyLjg4LDIuODgsMi44OHMyLjg4LTEuMjksMi44OC0yLjg4UzIwLjA4LDE1LjYyLDE4LjUsMTUuNjJ6Ii8+CjwvZz4KPC9zdmc+Cg=="
	grafanaInitPluginImage = "quay.io/grafana-operator/grafana_plugins_init:0.1.0"
	grafanaOauthProxyImage = "registry.redhat.io/openshift4/ose-oauth-proxy@sha256:582fc2d21cb3654f22f3ca50c39966041846e16a1543fc35c6a83948a2fa6c40"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager config.ConfigReadWriter
	Config        *config.Grafana
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	recorder      record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductGrafana],
		string(integreatlyv1alpha1.VersionGrafana),
		string(integreatlyv1alpha1.OperatorVersionGrafana),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for grafana")
	}

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace + "-operator"
	productConfig, err := configManager.ReadGrafana()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve grafana config: %w", err)
	}

	productConfig.SetNamespace(ns)
	productConfig.SetOperatorNamespace(productConfig.GetNamespace())
	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, fmt.Errorf("error writing grafana config : %w", err)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        productConfig,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Grafana reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, client, productNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, client, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		if err = r.deleteConsoleLink(ctx, client); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, client, operatorNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSecrets(ctx, client, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, client, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.GrafanaSubscriptionName), err)
		return phase, err
	}

	phase, err = r.ReconcileCsvDeploymentsPriority(
		ctx,
		client,
		fmt.Sprintf("grafana-operator.v%s", integreatlyv1alpha1.OperatorVersionGrafana),
		r.Config.GetOperatorNamespace(),
		r.installation.Spec.PriorityClassName,
	)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile grafana-operator csv deployments priority class name", err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to create components", err)
		return phase, err
	}

	phase, err = r.reconcileHost(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile host", err)
		return phase, err
	}
	rateLimit := productConfig.GetRateLimitConfig()
	activeQuota := productConfig.GetActiveQuota()

	phase, err = r.reconcileGrafanaDashboards(ctx, client, rateLimitDashBoardName, rateLimit, activeQuota)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile grafana dashboard", err)
		return phase, err
	}

	if string(r.Config.GetProductVersion()) != string(integreatlyv1alpha1.VersionGrafana) {
		r.Config.SetProductVersion(string(integreatlyv1alpha1.VersionGrafana))
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing grafana config : %w", err)
		}
	}

	alertsReconciler := r.newAlertReconciler(r.log, r.installation.Spec.Type, config.GetOboNamespace(r.installation.Namespace))
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile grafana alerts", err)
		return phase, err
	}

	if err := r.reconcileConsoleLink(ctx, client); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-k8s-proxy",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, secret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(secret, installation)
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data["session_secret"] = []byte(r.populateSessionProxySecret())
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileGrafanaDashboards(ctx context.Context, serverClient k8sclient.Client, dashboard string, limitConfig marin3rconfig.RateLimitConfig, activeQuota string) (integreatlyv1alpha1.StatusPhase, error) {

	grafanaDB := &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboard,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, grafanaDB, func() error {
		grafanaDB.Labels = map[string]string{
			"monitoring-key": "customer",
		}

		grafanaDB.Spec = grafanav1alpha1.GrafanaDashboardSpec{
			Json: getCustomerMonitoringGrafanaRateLimitJSON(fmt.Sprintf("%d", limitConfig.RequestsPerUnit), activeQuota),
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if opRes != controllerutil.OperationResultNone {
		r.log.Infof("Operation result grafana dashboard", l.Fields{"grafanaDashboard": grafanaDB.Name, "result": opRes})
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) scaleDeployment(ctx context.Context, client k8sclient.Client, name string, namespace string, scaleValue int32) (integreatlyv1alpha1.StatusPhase, error) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		err := client.Get(ctx, k8sclient.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, deployment)
		if err != nil {
			return fmt.Errorf("failed to get DeploymentConfig %s in namespace %s with error: %s", deployment.Name, namespace, err)
		}

		deployment.Spec.Replicas = &scaleValue
		err = client.Update(ctx, deployment)
		return err
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling grafana custom resource")

	var annotations = map[string]string{}
	annotations["service.alpha.openshift.io/serving-cert-secret-name"] = "grafana-k8s-tls"

	var serviceAccountAnnotations = map[string]string{}
	serviceAccountAnnotations["serviceaccounts.openshift.io/oauth-redirectreference.primary"] = "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"grafana-route\"}}"

	grafana := &grafanav1alpha1.Grafana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	status, err := controllerutil.CreateOrUpdate(ctx, client, grafana, func() error {
		owner.AddIntegreatlyOwnerAnnotations(grafana, r.installation)
		grafana.Spec = grafanav1alpha1.GrafanaSpec{
			Config: grafanav1alpha1.GrafanaConfig{
				Log: &grafanav1alpha1.GrafanaConfigLog{
					Mode:  "console",
					Level: "warn",
				},
				Auth: &grafanav1alpha1.GrafanaConfigAuth{
					DisableLoginForm:   &[]bool{false}[0],
					DisableSignoutMenu: &[]bool{true}[0],
				},
				AuthBasic: &grafanav1alpha1.GrafanaConfigAuthBasic{
					Enabled: &[]bool{true}[0],
				},
				AuthAnonymous: &grafanav1alpha1.GrafanaConfigAuthAnonymous{
					Enabled: &[]bool{true}[0],
				},
			},
			BaseImage: fmt.Sprintf("%s:%s", constants.GrafanaImage, constants.GrafanaVersion),
			InitImage: grafanaInitPluginImage,
			Containers: []v1.Container{
				{
					Name:  "grafana-proxy",
					Image: grafanaOauthProxyImage,
					VolumeMounts: []v1.VolumeMount{
						{MountPath: "/etc/tls/private",
							Name:     "secret-grafana-k8s-tls",
							ReadOnly: false,
						},
						{MountPath: "/etc/proxy/secrets",
							Name:     "secret-grafana-k8s-proxy",
							ReadOnly: false,
						},
					},
					Args: []string{
						"-provider=openshift",
						"-pass-basic-auth=false",
						"-https-address=:9091",
						"-http-address=",
						"-email-domain=*",
						"-upstream=http://localhost:3000",
						"-openshift-sar={\"resource\":\"namespaces\",\"verb\":\"get\"}",
						"-openshift-delegate-urls={\"/\":{\"resource\":\"namespaces\",\"verb\":\"get\"}}",
						"-tls-cert=/etc/tls/private/tls.crt",
						"-tls-key=/etc/tls/private/tls.key",
						"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
						"-cookie-secret-file=/etc/proxy/secrets/session_secret",
						"-openshift-service-account=grafana-serviceaccount",
						"-openshift-ca=/etc/pki/tls/cert.pem",
						"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						"-skip-auth-regex=^/metrics"},
					Ports: []v1.ContainerPort{
						{ContainerPort: 9091,
							Name: "grafana-proxy"},
					},
				},
			},
			Deployment: &grafanav1alpha1.GrafanaDeployment{
				PriorityClassName: r.installation.Spec.PriorityClassName,
			},
			Secrets: []string{"grafana-k8s-tls", "grafana-k8s-proxy"},
			Service: &grafanav1alpha1.GrafanaService{
				Ports: []v1.ServicePort{
					{Name: "grafana-proxy",
						Port:     9091,
						Protocol: v1.ProtocolTCP,
					},
				},
				Annotations: annotations,
			},
			Ingress: &grafanav1alpha1.GrafanaIngress{
				Enabled:     true,
				TargetPort:  "grafana-proxy",
				Termination: "reencrypt",
			},
			Client: &grafanav1alpha1.GrafanaClient{
				PreferService: boolPtr(true),
			},
			ServiceAccount: &grafanav1alpha1.GrafanaServiceAccount{
				Skip:        boolPtr(true),
				Annotations: serviceAccountAnnotations,
			},
			DashboardLabelSelector: []*metav1.LabelSelector{
				{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "monitoring-key",
							Operator: "In",
							Values:   []string{"customer"},
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.log.Infof("Grafana CR: ", l.Fields{"status": status})

	observabilityConfig, err := r.ConfigManager.ReadObservability()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	prometheusService := &corev1.Service{}

	err = client.Get(ctx, k8sclient.ObjectKey{Name: observabilityConfig.GetPrometheusOverride(), Namespace: observabilityConfig.GetNamespace()}, prometheusService)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	var upstreamPort int32
	for _, port := range prometheusService.Spec.Ports {
		if port.Name == "upstream" {
			upstreamPort = port.Port
		}
	}
	url := fmt.Sprintf("http://%s.%s.svc:%d", prometheusService.Name, prometheusService.Namespace, upstreamPort)

	dataSourceCR := &grafanav1alpha1.GrafanaDataSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customer-prometheus",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	status, err = controllerutil.CreateOrUpdate(ctx, client, dataSourceCR, func() error {
		owner.AddIntegreatlyOwnerAnnotations(dataSourceCR, r.installation)

		dataSourceCR.Spec = grafanav1alpha1.GrafanaDataSourceSpec{
			Datasources: []grafanav1alpha1.GrafanaDataSourceFields{
				{
					Name:      "Prometheus",
					Access:    "proxy",
					Editable:  true,
					IsDefault: true,
					Type:      "prometheus",
					Url:       url,
					Version:   1,
				},
			},
		}

		dataSourceCR.Spec.Datasources[0].JsonData.TimeInterval = "5s"

		dataSourceCR.Spec.Name = "customer.yaml"
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.log.Infof("Grafana datasource: ", l.Fields{"status": status})

	return r.reconcileServiceAccount(ctx, client)
}

func (r *Reconciler) reconcileServiceAccount(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	r.log.Info("Reconciling Grafana ServiceAccount")
	grafanaServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-serviceaccount",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, grafanaServiceAccount, func() error {
		serviceAccountAnnotations := grafanaServiceAccount.ObjectMeta.GetAnnotations()
		if serviceAccountAnnotations == nil {
			serviceAccountAnnotations = map[string]string{}
		}
		serviceAccountAnnotations["serviceaccounts.openshift.io/oauth-redirectreference.primary"] = "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"grafana-route\"}}"
		grafanaServiceAccount.ObjectMeta.SetAnnotations(serviceAccountAnnotations)

		return nil
	})
	if err != nil {
		r.log.Error("Failed reconciling grafana service account", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update grafana service account: %w", err)
	}
	r.log.Infof("Operation result on service account", l.Fields{"result": or})

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, _ *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling subscription")

	target := marketplace.Target{
		SubscriptionName: constants.GrafanaSubscriptionName,
		Namespace:        operatorNamespace,
	}

	catalogSourceReconciler, err := r.GetProductDeclaration().PrepareTarget(
		r.log,
		serverClient,
		marketplace.CatalogSourceName,
		&target,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{productNamespace},
		r.preUpgradeBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	return backup.NewNoopBackupExecutor()
}

// PopulateSessionProxySecret generates a session secret
func (r *Reconciler) populateSessionProxySecret() string {
	p, err := generatePassword(43)
	if err != nil {
		r.log.Error("Error executing PopulateSessionProxySecret", err)
	}
	return p
}

// GeneratePassword returns a base64 encoded securely random bytes.
func generatePassword(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), err
}

func (r *Reconciler) reconcileHost(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	grafanaRoute := &routev1.Route{}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultRoutename, Namespace: r.Config.GetOperatorNamespace()}, grafanaRoute)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to get route for Grafana: %w", err)
	}

	r.Config.SetHost("https://" + grafanaRoute.Spec.Host)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Could not set Grafana route: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func GetGrafanaConsoleURL(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (string, error) {

	grafanaConsoleURL := installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductGrafana].Host
	if grafanaConsoleURL != "" {
		return grafanaConsoleURL, nil
	}

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	grafanaRoute := &routev1.Route{}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultRoutename, Namespace: ns}, grafanaRoute)
	if err != nil {
		return "", err
	}

	return "https://" + grafanaRoute.Spec.Host, nil
}

func (r *Reconciler) reconcileConsoleLink(ctx context.Context, serverClient k8sclient.Client) error {
	// If the installation type isn't managed-api, ensure that the ConsoleLink
	// doesn't exist
	if integreatlyv1alpha1.IsRHOAMSingletenant(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {

		cl := &consolev1.ConsoleLink{
			ObjectMeta: metav1.ObjectMeta{
				Name: grafanaConsoleLink,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cl, func() error {
			cl.Spec = consolev1.ConsoleLinkSpec{
				ApplicationMenu: &consolev1.ApplicationMenuSpec{
					ImageURL: grafanaIcon,
					Section:  "OpenShift Managed Services",
				},
				Location: consolev1.ApplicationMenu,
				Link: consolev1.Link{
					Href: r.Config.GetHost(),
					Text: "API Management Dashboards",
				},
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("error reconciling console link: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) deleteConsoleLink(ctx context.Context, serverClient k8sclient.Client) error {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: grafanaConsoleLink,
		},
	}

	err := serverClient.Delete(ctx, cl)
	if err != nil && !k8serr.IsNotFound(err) {
		return fmt.Errorf("error removing grafana console link: %v", err)
	}

	return nil
}

func boolPtr(value bool) *bool {
	return &value
}
