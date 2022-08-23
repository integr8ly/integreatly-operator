/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"

	integreatlymetrics "github.com/integr8ly/integreatly-operator/pkg/metrics"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	namespacecontroller "github.com/integr8ly/integreatly-operator/controllers/namespacelabel"
	rhmicontroller "github.com/integr8ly/integreatly-operator/controllers/rhmi"
	rhmiconfigcontroller "github.com/integr8ly/integreatly-operator/controllers/rhmiconfig"
	subscriptioncontroller "github.com/integr8ly/integreatly-operator/controllers/subscription"
	tenantcontroller "github.com/integr8ly/integreatly-operator/controllers/tenant"
	usercontroller "github.com/integr8ly/integreatly-operator/controllers/user"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/webhooks"
	"github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// Register custom metrics with the global prometheus registry
	customMetrics.Registry.MustRegister(integreatlymetrics.OperatorVersion)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIStatusAvailable)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIInfo)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIVersion)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIStatus)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHOAMVersion)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHOAMStatus)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHOAMAlertsSummary)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHOAMCluster)
	customMetrics.Registry.MustRegister(integreatlymetrics.ThreeScaleUserAction)
	customMetrics.Registry.MustRegister(integreatlymetrics.Quota)
	customMetrics.Registry.MustRegister(integreatlymetrics.TotalNumTenants)
	customMetrics.Registry.MustRegister(integreatlymetrics.NumReconciledTenants)
	customMetrics.Registry.MustRegister(integreatlymetrics.NumFailedTenants)
	customMetrics.Registry.MustRegister(integreatlymetrics.NoActivated3ScaleTenantAccount)
	customMetrics.Registry.MustRegister(integreatlymetrics.InstallationControllerReconcileDelayed)
	customMetrics.Registry.MustRegister(integreatlymetrics.CustomDomain)

	integreatlymetrics.OperatorVersion.Add(1)
	utilruntime.Must(v1.Install(clientgoscheme.Scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(rhmiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(rhmiv1alpha1.AddToSchemes.AddToScheme(scheme))
	utilruntime.Must(apiextensions.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
	// +kubebuilder:rbac:groups=integreatly.org,resources=apimanagementtenant,verbs=watch;get;list

}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8383", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    true,
		QuoteEmptyFields: false,
	})

	watchNamespace, err := k8s.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get WatchNamespace, "+
			"the manager will watch and manage resources in all namespaces")
	}

	var mgr ctrl.Manager
	if strings.Contains(watchNamespace, "sandbox") || watchNamespace == "" {
		mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     metricsAddr,
			Port:                   9443,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "28185cee.integreatly.org",
		})
		if err != nil {
			setupLog.Error(err, "unable to start multitenant manager")
			os.Exit(1)
		}
	} else {
		mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     metricsAddr,
			Port:                   9443,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "28185cee.integreatly.org",
			Namespace:              watchNamespace,
		})
		if err != nil {
			setupLog.Error(err, "unable to start singletenant manager")
			os.Exit(1)
		}
	}

	if err = rhmicontroller.New(mgr).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RHMI")
		os.Exit(1)
	}
	if err = (&rhmiconfigcontroller.RHMIConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RHMIConfig")
		os.Exit(1)
	}
	if watchNamespace == "" || !strings.Contains(watchNamespace, "sandbox") {
		if err = namespacecontroller.New(mgr).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Namespace")
			os.Exit(1)
		}
		if err = (&usercontroller.UserReconciler{}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "User")
			os.Exit(1)
		}
	}

	if strings.Contains(watchNamespace, "sandbox") {
		tenantCtrl, err := tenantcontroller.New(mgr)
		if err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TenantController")
			os.Exit(1)
		}
		if err = tenantCtrl.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to setup controller", "controller", "TenantController")
			os.Exit(1)
		}
	}

	subscriptionCtrl, err := subscriptioncontroller.New(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Subscription")
		os.Exit(1)
	}
	if err = subscriptionCtrl.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup controller", "controller", "Subscription")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := setupWebhooks(mgr); err != nil {
		setupLog.Error(err, "Error setting up webhook server")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupWebhooks(mgr ctrl.Manager) error {
	rhmiConfigRegister, err := webhooks.WebhookRegisterFor(&rhmiv1alpha1.RHMIConfig{})
	if err != nil {
		return err
	}

	webhooks.Config.AddWebhook(webhooks.IntegreatlyWebhook{
		Name:     "rhmiconfig",
		Register: rhmiConfigRegister,
		Rule: webhooks.NewRule().
			OneResource("integreatly.org", "v1alpha1", "rhmiconfigs").
			ForCreate().
			ForUpdate().
			NamespacedScope(),
	})

	webhooks.Config.AddWebhook(webhooks.IntegreatlyWebhook{
		Name: "rhmiconfig-mutate",
		Rule: webhooks.NewRule().
			OneResource("integreatly.org", "v1alpha1", "rhmiconfigs").
			ForCreate().
			ForUpdate().
			NamespacedScope(),
		Register: webhooks.AdmissionWebhookRegister{
			Type: webhooks.MutatingType,
			Path: "/mutate-rhmiconfig",
			Hook: &admission.Webhook{
				Handler: rhmiv1alpha1.NewRHMIConfigMutatingHandler(),
			},
		},
	})

	// Delete webhook for the RHMI CR that uninstalls the operator if there
	// are no finalizers left
	webhooks.Config.AddWebhook(webhooks.IntegreatlyWebhook{
		Name: "rhmi-delete",
		Rule: webhooks.NewRule().
			OneResource("integreatly.org", "v1alpha1", "rhmis").
			ForDelete().
			NamespacedScope(),
		Register: webhooks.AdmissionWebhookRegister{
			Type: webhooks.ValidatingType,
			Path: "/delete-rhmi",
			Hook: &admission.Webhook{
				Handler: addon.NewDeleteRHMIHandler(mgr.GetConfig(), mgr.GetScheme()),
			},
		},
	})

	// The webhooks feature can't work when the operator runs locally, as it
	// needs to be accessible by kubernetes and depends on the TLS certificates
	// being mounted
	webhooks.Config.Enabled = k8s.IsRunInCluster()

	if err := webhooks.Config.SetupServer(mgr); err != nil {
		return err
	}

	return nil
}
