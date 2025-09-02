package grafana

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "customer-monitoring"
	defaultRoutename             = "grafana-route"
	grafanaDeployment            = "grafana-deployment"
	grafanaConsoleLink           = "grafana-user-console-link"
	grafanaIcon                  = "data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI1LjIuMCwgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAzNyAzNyIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMzcgMzc7IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KCS5zdDB7ZmlsbDojRUUwMDAwO30KCS5zdDF7ZmlsbDojRkZGRkZGO30KPC9zdHlsZT4KPGc+Cgk8cGF0aCBkPSJNMjcuNSwwLjVoLTE4Yy00Ljk3LDAtOSw0LjAzLTksOXYxOGMwLDQuOTcsNC4wMyw5LDksOWgxOGM0Ljk3LDAsOS00LjAzLDktOXYtMThDMzYuNSw0LjUzLDMyLjQ3LDAuNSwyNy41LDAuNUwyNy41LDAuNXoiCgkJLz4KCTxnPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yNSwyMi4zN2MtMC45NSwwLTEuNzUsMC42My0yLjAyLDEuNWgtMS44NVYyMS41YzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYycy0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDIuNDhjMC4yNywwLjg3LDEuMDcsMS41LDIuMDIsMS41YzEuMTcsMCwyLjEyLTAuOTUsMi4xMi0yLjEyUzI2LjE3LDIyLjM3LDI1LDIyLjM3eiBNMjUsMjUuMzcKCQkJYy0wLjQ4LDAtMC44OC0wLjM5LTAuODgtMC44OHMwLjM5LTAuODgsMC44OC0wLjg4czAuODgsMC4zOSwwLjg4LDAuODhTMjUuNDgsMjUuMzcsMjUsMjUuMzd6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTIwLjUsMTYuMTJjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTIuMzhoMS45MWMwLjMyLDAuNzcsMS4wOCwxLjMxLDEuOTYsMS4zMQoJCQljMS4xNywwLDIuMTItMC45NSwyLjEyLTIuMTJzLTAuOTUtMi4xMi0yLjEyLTIuMTJjLTEuMDIsMC0xLjg4LDAuNzMtMi4wOCwxLjY5SDIwLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJQzE5Ljg3LDE1Ljg1LDIwLjE2LDE2LjEyLDIwLjUsMTYuMTJ6IE0yNSwxMS40M2MwLjQ4LDAsMC44OCwwLjM5LDAuODgsMC44OHMtMC4zOSwwLjg4LTAuODgsMC44OHMtMC44OC0wLjM5LTAuODgtMC44OAoJCQlTMjQuNTIsMTEuNDMsMjUsMTEuNDN6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTEyLjEyLDE5Ljk2di0wLjg0aDIuMzhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJzLTAuMjgtMC42Mi0wLjYyLTAuNjJoLTIuMzh2LTAuOTEKCQkJYzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYyaC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYzYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDNDMTEuODQsMjAuNTksMTIuMTIsMjAuMzEsMTIuMTIsMTkuOTYKCQkJeiBNMTAuODcsMTkuMzRIOS4xMnYtMS43NWgxLjc1VjE5LjM0eiIvPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yOC41LDE2LjM0aC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYwLjkxSDIyLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYyczAuMjgsMC42MiwwLjYyLDAuNjJoMi4zOAoJCQl2MC44NGMwLDAuMzUsMC4yOCwwLjYyLDAuNjIsMC42MmgzYzAuMzQsMCwwLjYyLTAuMjgsMC42Mi0wLjYydi0zQzI5LjEyLDE2LjYyLDI4Ljg0LDE2LjM0LDI4LjUsMTYuMzR6IE0yNy44NywxOS4zNGgtMS43NXYtMS43NQoJCQloMS43NVYxOS4zNHoiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwyMC44N2MtMC4zNCwwLTAuNjMsMC4yOC0wLjYzLDAuNjJ2Mi4zOGgtMS44NWMtMC4yNy0wLjg3LTEuMDctMS41LTIuMDItMS41CgkJCWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMmMwLjk1LDAsMS43NS0wLjYzLDIuMDItMS41aDIuNDhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTMKCQkJQzE3LjEyLDIxLjE1LDE2Ljg0LDIwLjg3LDE2LjUsMjAuODd6IE0xMiwyNS4zN2MtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4CgkJCVMxMi40OCwyNS4zNywxMiwyNS4zN3oiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwxMS44N2gtMi40MmMtMC4yLTAuOTctMS4wNi0xLjY5LTIuMDgtMS42OWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMgoJCQljMC44OCwwLDEuNjQtMC41NCwxLjk2LTEuMzFoMS45MXYyLjM4YzAsMC4zNSwwLjI4LDAuNjIsMC42MywwLjYyczAuNjItMC4yOCwwLjYyLTAuNjJ2LTNDMTcuMTIsMTIuMTUsMTYuODQsMTEuODcsMTYuNSwxMS44N3oKCQkJIE0xMiwxMy4xOGMtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4UzEyLjQ4LDEzLjE4LDEyLDEzLjE4eiIvPgoJPC9nPgoJPHBhdGggY2xhc3M9InN0MSIgZD0iTTE4LjUsMjIuNjJjLTIuMjcsMC00LjEzLTEuODUtNC4xMy00LjEyczEuODUtNC4xMiw0LjEzLTQuMTJzNC4xMiwxLjg1LDQuMTIsNC4xMlMyMC43NywyMi42MiwxOC41LDIyLjYyegoJCSBNMTguNSwxNS42MmMtMS41OCwwLTIuODgsMS4yOS0yLjg4LDIuODhzMS4yOSwyLjg4LDIuODgsMi44OHMyLjg4LTEuMjksMi44OC0yLjg4UzIwLjA4LDE1LjYyLDE4LjUsMTUuNjJ6Ii8+CjwvZz4KPC9zdmc+Cg=="
	gfSecurityAdminUser          = "admin"
	ratelimitConfigMapName       = "ratelimit-grafana-dashboard"
	ratelimitCMDataKey           = "ratelimit.json"
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
	// Stub. Grafana installed by Package Operator
	return true
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger) (*Reconciler, error) {
	productConfig, err := configManager.ReadGrafana()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve grafana config: %w", err)
	}

	productConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, fmt.Errorf("error writing grafana config : %w", err)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        productConfig,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=list;get;watch;update;delete

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Grafana reconcile")
	if resources.IsInProw(installation) {
		r.log.Info("Running in prow, skipping Grafana reconciliation")
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	productNamespace := r.Config.GetNamespace()
	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, client, productNamespace, r.log)
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

	phase, err = r.reconcileSecrets(ctx, client, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	requestsPerUnitStr := fmt.Sprint(productConfig.GetRateLimitConfig().RequestsPerUnit)
	activeQuota := productConfig.GetActiveQuota()

	// Creates Grafana RateLimit ConfigMap
	phase, err = r.ReconcileGrafanaRateLimitDashboardConfigMap(ctx, client, requestsPerUnitStr, activeQuota)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile Grafana Ratelimit Dashboard ConfigMap", err)
		return phase, err
	}

	phase, err = r.ReconcileGrafanaDeployment(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile Grafana Deployment", err)
		return phase, err
	}

	deploymentDone, err := r.checkDeploymentStatus(client)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to check Grafana Deployment status", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if !deploymentDone {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if string(r.Config.GetProductVersion()) != string(integreatlyv1alpha1.VersionGrafana) {
		r.Config.SetProductVersion(string(integreatlyv1alpha1.VersionGrafana))
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing grafana config : %w", err)
		}
	}

	err = r.removeGrafanaOperatorAlerts(r.installation.Spec.NamespacePrefix, ctx, client)
	if err != nil {
		r.log.Error("Error removing obsolete Grafana Operator alerts: ", err)
	}
	alertsReconciler := r.newAlertReconciler(r.log, r.installation.Spec.Type, productNamespace)
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, client); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile grafana alerts", err)
		return phase, err
	}

	phase, err = r.reconcileHost(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile host", err)
		return phase, err
	}
	if err := r.reconcileConsoleLink(ctx, client); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()

	if err := r.deleteMonitoringOperatorNamespace(ctx, client); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcileGrafanaRateLimitDashboardConfigMap(ctx context.Context, client k8sclient.Client, requestsPerUnit string, activeQuota string) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()
	log.Info("reconciling Grafana RateLimit Dashboard ConfigMap")

	rateLimitConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ratelimitConfigMapName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, rateLimitConfigMap, func() error {
		if rateLimitConfigMap.Data == nil {
			rateLimitConfigMap.Data = map[string]string{}
		}
		ratelimitJsonStr := getCustomerMonitoringGrafanaRateLimitJSON(requestsPerUnit, activeQuota)
		rateLimitConfigMap.Data[ratelimitCMDataKey] = ratelimitJsonStr

		return nil

	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ReconcileGrafanaDeployment(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()
	log.Info("reconciling Grafana Deployment")

	grafanaDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-deployment",
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, grafanaDeployment, func() error {
		if grafanaDeployment.Labels == nil {
			grafanaDeployment.Labels = map[string]string{}
		}

		if grafanaDeployment.Annotations == nil {
			grafanaDeployment.Annotations = map[string]string{}
		}

		grafanaDeployment.Labels["app"] = "grafana"
		grafanaDeployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "grafana",
			},
		}

		grafanaDeployment.Spec.Replicas = func(i int32) *int32 { return &i }(1)

		grafanaDeployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "grafana",
			},
			Annotations: map[string]string{
				"prometheus.io/port":   "3000",
				"prometheus.io/scrape": "true",
			},
		}

		grafanaDeployment.Spec.Template.Spec.PriorityClassName = r.installation.Spec.PriorityClassName
		if grafanaDeployment.Spec.Template.Spec.Containers == nil {
			grafanaDeployment.Spec.Template.Spec.Containers = []corev1.Container{{}, {}}
		}

		grafanaDeployment.Spec.Template.Spec.ServiceAccountName = "grafana-serviceaccount"

		grafanaDeployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "grafana-provision-plugins",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "grafana-provision-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "grafana-provision-config",
						},
					},
				},
			},
			{
				Name: "grafana-provision-notifiers",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "grafana-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "grafana-config",
						},
					},
				},
			},
			{
				Name: "grafana-logs",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "grafana-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "grafana-plugins",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "grafana-datasources",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "grafana-datasources",
						},
					},
				},
			},
			{
				Name: "ratelimit-grafana-dashboard",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "ratelimit-grafana-dashboard",
						},
					},
				},
			},
			{
				Name: "secret-grafana-k8s-tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "grafana-k8s-tls",
						Optional:   func(b bool) *bool { return &b }(true),
					},
				},
			},
			{
				Name: "secret-grafana-k8s-proxy",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "grafana-k8s-proxy",
						Optional:   func(b bool) *bool { return &b }(true),
					},
				},
			},
		}

		// Container #1
		grafanaDeployment.Spec.Template.Spec.Containers[0].TerminationMessagePath = "/dev/termination-log"
		grafanaDeployment.Spec.Template.Spec.Containers[0].Name = "grafana"
		grafanaDeployment.Spec.Template.Spec.Containers[0].Image = "registry.redhat.io/rhel9/grafana:9.6-1755762368"
		grafanaDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				MountPath: "/etc/grafana/",
				Name:      "grafana-config",
			},
			{
				MountPath: "/var/lib/grafana",
				Name:      "grafana-data",
			},
			{
				MountPath: "/var/lib/grafana/plugins",
				Name:      "grafana-plugins",
			},
			{
				MountPath: "/etc/grafana/provisioning/plugins",
				Name:      "grafana-provision-plugins",
			},
			{
				MountPath: "/etc/grafana/provisioning/dashboards",
				Name:      "grafana-provision-config",
			},
			{
				MountPath: "/etc/grafana/provisioning/notifiers",
				Name:      "grafana-provision-notifiers",
			},
			{
				MountPath: "/var/log/grafana",
				Name:      "grafana-logs",
			},
			{
				MountPath: "/etc/grafana/provisioning/datasources",
				Name:      "grafana-datasources",
			},
			{
				MountPath: "/var/lib/grafana/dashboards",
				Name:      "ratelimit-grafana-dashboard",
			},
			{
				MountPath: "/etc/grafana-secrets/grafana-k8s-tls",
				Name:      "secret-grafana-k8s-tls",
			},
			{
				MountPath: "/etc/grafana-secrets/grafana-k8s-proxy",
				Name:      "secret-grafana-k8s-proxy",
			},
		}
		grafanaDeployment.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          "grafana-http",
				ContainerPort: 3000,
			},
		}

		grafanaDeployment.Spec.Template.Spec.Containers[0].Args = []string{"-config=/etc/grafana/grafana.ini"}

		grafanaDeployment.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name: "LAST_CONFIG",
			},
			{
				Name: "LAST_DATASOURCES",
			},
			{
				Name: "GF_SECURITY_ADMIN_USER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "grafana-admin-credentials",
						},
						Key: "GF_SECURITY_ADMIN_USER",
					},
				},
			},
			{
				Name: "GF_SECURITY_ADMIN_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "grafana-admin-credentials",
						},
						Key: "GF_SECURITY_ADMIN_PASSWORD",
					},
				},
			},
		}

		grafanaDeployment.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/api/health",
					Port:   intstr.FromInt(3000),
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: 60,
			TimeoutSeconds:      30,
			FailureThreshold:    10,
		}

		grafanaDeployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/api/health",
					Port:   intstr.FromInt(3000),
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      3,
			FailureThreshold:    1,
		}

		grafanaDeployment.Spec.Template.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		}

		grafanaDeployment.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		}

		// container #2
		grafanaDeployment.Spec.Template.Spec.Containers[1].TerminationMessagePath = "/dev/termination-log"
		grafanaDeployment.Spec.Template.Spec.Containers[1].Name = "grafana-proxy"
		grafanaDeployment.Spec.Template.Spec.Containers[1].Image = "registry.redhat.io/openshift4/ose-oauth-proxy-rhel9:v4.15.0-202508190116.p2.g241a88c.assembly.stream.el9"
		grafanaDeployment.Spec.Template.Spec.Containers[1].VolumeMounts = []corev1.VolumeMount{
			{
				MountPath: "/etc/tls/private",
				Name:      "secret-grafana-k8s-tls",
			},
			{
				MountPath: "/etc/proxy/secrets",
				Name:      "secret-grafana-k8s-proxy",
			},
		}
		grafanaDeployment.Spec.Template.Spec.Containers[1].Ports = []corev1.ContainerPort{
			{
				Name:          "grafana-proxy",
				ContainerPort: 9091,
			},
		}

		grafanaDeployment.Spec.Template.Spec.Containers[1].Args = []string{
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
			"-skip-auth-regex=^/metrics",
		}

		return nil

	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileHost(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	grafanaRoute := &routev1.Route{}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultRoutename, Namespace: r.Config.GetNamespace()}, grafanaRoute)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get route for Grafana: %w", err)
	}

	r.Config.SetHost("https://" + grafanaRoute.Spec.Host)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not set Grafana route: %w", err)
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

func (r *Reconciler) deleteMonitoringOperatorNamespace(ctx context.Context, serverClient k8sclient.Client) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace() + "-operator",
		},
	}
	err := serverClient.Delete(ctx, ns)
	if err != nil && !k8serr.IsNotFound(err) {
		return fmt.Errorf("error removing unused customer-monitoring-operator namespace: %v", err)
	}
	return nil
}

func (r *Reconciler) reconcileSecrets(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()
	log.Info("reconciling Grafana configuration secrets")

	grafanaProxySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-k8s-proxy",
			Namespace: r.Config.GetNamespace(),
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, grafanaProxySecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(grafanaProxySecret, installation)
		if grafanaProxySecret.Data == nil {
			grafanaProxySecret.Data = map[string][]byte{}
		}
		grafanaProxySecret.Data["session_secret"] = []byte(resources.GenerateRandomPassword(20, 2, 2, 2))
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	grafanaAdminCredsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-admin-credentials",
			Namespace: r.Config.GetNamespace(),
		},
		Data: getAdminCredsSecretData(),
		Type: corev1.SecretTypeOpaque,
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, grafanaAdminCredsSecret, func() error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getAdminCredsSecretData() map[string][]byte {
	password := []byte(RandStringRunes(10))
	credentials := map[string][]byte{
		"GF_SECURITY_ADMIN_USER":     []byte(gfSecurityAdminUser),
		"GF_SECURITY_ADMIN_PASSWORD": password,
	}

	// Make the credentials available to the environment, similar is it was done in Grafana operator (resolve admin login issue?)
	err := os.Setenv("GF_SECURITY_ADMIN_USER", string(credentials["GF_SECURITY_ADMIN_USER"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars")
		return credentials
	}
	err = os.Setenv("GF_SECURITY_ADMIN_PASSWORD", string(credentials["GF_SECURITY_ADMIN_PASSWORD"]))
	if err != nil {
		fmt.Printf("can't set credentials as environment vars (optional)")
		return credentials
	}

	return credentials
}

func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func RandStringRunes(s int) string {
	b := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}

func (r *Reconciler) checkDeploymentStatus(client k8sclient.Client) (bool, error) {
	deployment := &appsv1.Deployment{}
	if err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: grafanaDeployment, Namespace: r.Config.GetNamespace()}, deployment); err != nil {
		return false, err
	}
	if deployment.Status.AvailableReplicas == 0 || deployment.Status.ReadyReplicas < deployment.Status.Replicas {
		return false, nil
	}
	return true, nil
}
