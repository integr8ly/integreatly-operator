package grafana

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	consolev1 "github.com/openshift/api/console/v1"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "customer-monitoring"
	manifestPackage              = "integreatly-grafana"
	defaultGrafanaName           = "grafana"
	defaultRoutename             = defaultGrafanaName + "-route"
	rateLimitDashBoardName       = "rate-limit"

	grafanaConsoleLink = "grafana-user-console-link"
	grafanaIcon        = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAgMTAwIj48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2Q3MWUwMDt9LmNscy0ye2ZpbGw6I2MyMWEwMDt9LmNscy0ze2ZpbGw6I2ZmZjt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPnByb2R1Y3RpY29uc18xMDE3X1JHQl9BUEkgZmluYWwgY29sb3I8L3RpdGxlPjxnIGlkPSJMYXllcl8xIiBkYXRhLW5hbWU9IkxheWVyIDEiPjxjaXJjbGUgY2xhc3M9ImNscy0xIiBjeD0iNTAiIGN5PSI1MCIgcj0iNTAiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0yMC43MSA1MCkgcm90YXRlKC00NSkiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik04NS4zNiwxNC42NEE1MCw1MCwwLDAsMSwxNC42NCw4NS4zNloiLz48cGF0aCBjbGFzcz0iY2xzLTMiIGQ9Ik01MC4yNSwzMC44M2EyLjY5LDIuNjksMCwxLDAtMi42OC0yLjY5QTIuNjUsMi42NSwwLDAsMCw1MC4yNSwzMC44M1pNNDMuMzYsMzkuNGEzLjM1LDMuMzUsMCwwLDAsMy4zMiwzLjM0LDMuMzQsMy4zNCwwLDAsMCwwLTYuNjdBMy4zNSwzLjM1LDAsMCwwLDQzLjM2LDM5LjRabTMuOTIsOS44OUEyLjY4LDIuNjgsMCwxLDAsNDQuNiw1MiwyLjcsMi43LDAsMCwwLDQ3LjI4LDQ5LjI5Wk0zMi42MywyOS42NWEzLjI2LDMuMjYsMCwxLDAtMy4yNC0zLjI2QTMuMjYsMy4yNiwwLDAsMCwzMi42MywyOS42NVpNNDAuNTMsMzRhMi43NywyLjc3LDAsMCwwLDAtNS41MywyLjc5LDIuNzksMCwwLDAtMi43NiwyLjc3QTIuODUsMi44NSwwLDAsMCw0MC41MywzNFptMS43Ni05LjMxYTQuNCw0LjQsMCwxLDAtNC4zOC00LjRBNC4zNyw0LjM3LDAsMCwwLDQyLjI5LDI0LjcxWk0zMi43OCw0OWE3LDcsMCwxLDAtNy03QTcsNywwLDAsMCwzMi43OCw0OVptMzIuMTMtNy43YTQuMjMsNC4yMywwLDAsMCw0LjMsNC4zMSw0LjMxLDQuMzEsMCwxLDAtNC4zLTQuMzFabTYuOSwxMC4wNmEzLjA4LDMuMDgsMCwxLDAsMy4wOC0zLjA5QTMuMDksMy4wOSwwLDAsMCw3MS44MSw1MS4zOFpNNzMuOSwzNC43N2E0LjMxLDQuMzEsMCwxLDAtNC4zLTQuMzFBNC4yOCw0LjI4LDAsMCwwLDczLjksMzQuNzdaTTUyLjE2LDQ1LjA2YTMuNjUsMy42NSwwLDEsMCwzLjY1LTMuNjZBMy42NCwzLjY0LDAsMCwwLDUyLjE2LDQ1LjA2Wk01NSwyMmEzLjE3LDMuMTcsMCwwLDAsMy4xNi0zLjE3QTMuMjMsMy4yMywwLDAsMCw1NSwxNS42MywzLjE3LDMuMTcsMCwwLDAsNTUsMjJabS0uNDcsMTAuMDlBNS4zNyw1LjM3LDAsMCwwLDYwLDM3LjU0YTUuNDgsNS40OCwwLDEsMC01LjQ1LTUuNDhaTTY2LjI1LDI1LjVhMi42OSwyLjY5LDAsMSwwLTIuNjgtMi42OUEyLjY1LDIuNjUsMCwwLDAsNjYuMjUsMjUuNVpNNDUuNyw2My4xYTMuNDIsMy40MiwwLDEsMC0zLjQxLTMuNDJBMy40MywzLjQzLDAsMCwwLDQ1LjcsNjMuMVptMTQsMTEuMTlhNC40LDQuNCwwLDEsMCw0LjM4LDQuNEE0LjM3LDQuMzcsMCwwLDAsNTkuNzMsNzQuMjlaTTYyLjMsNTAuNTFhOS4yLDkuMiwwLDEsMCw5LjE2LDkuMkE5LjIyLDkuMjIsMCwwLDAsNjIuMyw1MC41MVpNNTAuMSw2Ni43N2EyLjY5LDIuNjksMCwxLDAsMi42OCwyLjY5QTIuNywyLjcsMCwwLDAsNTAuMSw2Ni43N1pNODEuMjUsNDEuMTJhMi43LDIuNywwLDAsMC0yLjY4LDIuNjksMi42NSwyLjY1LDAsMCwwLDIuNjgsMi42OSwyLjY5LDIuNjksMCwwLDAsMC01LjM3Wk00NC40OSw3Ni40N2EzLjczLDMuNzMsMCwwLDAtMy43MywzLjc0LDMuNzcsMy43NywwLDEsMCwzLjczLTMuNzRaTTc5LjA2LDU2LjcyYTQsNCwwLDEsMCw0LDRBNCw0LDAsMCwwLDc5LjA2LDU2LjcyWm0tNiwxMS43OEEzLjA5LDMuMDksMCwwLDAsNzAsNzEuNmEzLDMsMCwwLDAsMy4wOCwzLjA5LDMuMDksMy4wOSwwLDAsMCwwLTYuMTlaTTI4LjMsNjhhNC4xNiw0LjE2LDAsMCwwLTQuMTQsNC4xNUE0LjIxLDQuMjEsMCwwLDAsMjguMyw3Ni4zYTQuMTUsNC4xNSwwLDAsMCwwLTguM1ptLTguMjItOWEzLDMsMCwxLDAsMywzQTMuMDUsMy4wNSwwLDAsMCwyMC4wOCw1OVptMS44NC05Ljc0YTMsMywwLDEsMCwzLDNBMy4wNSwzLjA1LDAsMCwwLDIxLjkxLDQ5LjIyWk0yMi4zNyw0MmEzLjI0LDMuMjQsMCwxLDAtMy4yNCwzLjI2QTMuMjYsMy4yNiwwLDAsMCwyMi4zNyw0MlpNNDMuMTEsNzAuMmEzLjgsMy44LDAsMCwwLTMuODEtMy43NCwzLjczLDMuNzMsMCwwLDAtMy43MywzLjc0QTMuOCwzLjgsMCwwLDAsMzkuMyw3NCwzLjg3LDMuODcsMCwwLDAsNDMuMTEsNzAuMlpNMzcuNTYsNTguNDNhNC42OCw0LjY4LDAsMCwwLTQuNjItNC42NCw0LjYzLDQuNjMsMCwwLDAtNC42Miw0LjY0LDQuNTgsNC41OCwwLDAsMCw0LjYyLDQuNjRBNC42Myw0LjYzLDAsMCwwLDM3LjU2LDU4LjQzWk0yMy4xMSwzMy44MmEyLjUyLDIuNTIsMCwxLDAtMi41MS0yLjUyQTIuNTMsMi41MywwLDAsMCwyMy4xMSwzMy44MloiLz48L2c+PC9zdmc+"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager config.ConfigReadWriter
	Config        *config.Grafana
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	extraParams   map[string]string
	recorder      record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductGrafana],
		string(integreatlyv1alpha1.VersionGrafana),
		string(integreatlyv1alpha1.OperatorVersionGrafana),
	)
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for grafana")
	}

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace + "-operator"
	config, err := configManager.ReadGrafana()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve grafana config: %w", err)
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(ns)
		configManager.WriteConfig(config)
	}
	if config.GetOperatorNamespace() == "" {
		config.SetOperatorNamespace(config.GetNamespace())
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, productConfig quota.ProductConfig) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Grafana reconcile")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, client, productNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, client, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		if err := r.deleteConsoleLink(ctx, client); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, client, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSecrets(ctx, client, installation, &grafanav1alpha1.Grafana{})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, client, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.ThreeScaleSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, client, installation)
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
		r.ConfigManager.WriteConfig(r.Config)
	}

	alertsReconciler := r.newAlertReconciler(r.log, r.installation.Spec.Type)
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile grafana alerts", err)
		return phase, err
	}

	if err := r.reconcileConsoleLink(ctx, client); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI, cr *grafanav1alpha1.Grafana) (integreatlyv1alpha1.StatusPhase, error) {
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

func (r *Reconciler) reconcileComponents(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
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
			Containers: []v1.Container{
				{Name: "grafana-proxy",
					Image: "quay.io/openshift/origin-oauth-proxy:4.2",
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
				PreferService: true,
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

	prometheusNamespace := fmt.Sprintf("%smiddleware-monitoring-operator", installation.Spec.NamespacePrefix)

	prometheusService := &corev1.Service{}

	err = client.Get(ctx, k8sclient.ObjectKey{Name: "prometheus-service", Namespace: prometheusNamespace}, prometheusService)
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

		spec := grafanav1alpha1.GrafanaDataSourceSpec{
			Datasources: []grafanav1alpha1.GrafanaDataSourceFields{
				{
					Name:      "Prometheus",
					Access:    "proxy",
					Editable:  true,
					IsDefault: true,
					JsonData: grafanav1alpha1.GrafanaDataSourceJsonData{
						TimeInterval: "5s",
					},
					Type:    "prometheus",
					Url:     url,
					Version: 1,
				},
			},
		}

		dataSourceCR.Spec = spec
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

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
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

	grafanaConsoleURL := installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductGrafana].Host
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
	if r.installation.Spec.Type != string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return nil
	}

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
