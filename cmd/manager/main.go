package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"

	"os"
	"runtime"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/apis"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller"
	integreatlymetrics "github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/webhooks"
	"github.com/integr8ly/integreatly-operator/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

var log = logf.Log.WithName("cmd")

func init() {
	// Register custom metrics with the global prometheus registry
	customMetrics.Registry.MustRegister(integreatlymetrics.OperatorVersion)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIStatusAvailable)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIInfo)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIVersion)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHMIStatus)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHOAMVersion)
	customMetrics.Registry.MustRegister(integreatlymetrics.RHOAMStatus)
	integreatlymetrics.OperatorVersion.Add(1)
}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.GetVersion()))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:      true,
		FullTimestamp:    true,
		QuoteEmptyFields: false,
	})
	if err := setLogLevel(); err != nil {
		// print as assume loggers may not work
		fmt.Printf("Failed to set log level %s ", err)
	}
	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()

	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "rhmi-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     apiutil.NewDiscoveryRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add monitoring resources
	if err := monitoringv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add the Metrics Service
	addMetrics(ctx, cfg)

	// Start up the wehook server
	if err := setupWebhooks(mgr); err != nil {
		log.Error(err, "Error setting up webhook server")
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

// setLogLevel will decide the log level based on the LOG_LEVEL env var valid values are (debug, info, warn and error)
func setLogLevel() error {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	if err := zap.FlagSet().Set("zap-level", logLevel); err != nil {
		return fmt.Errorf("failed to set zap log level to %s %s", logLevel, err)
	}
	logrusLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("failed to set logrus log level to %s %s", logLevel, err)
	}
	logrus.SetLevel(logrusLevel)
	return nil
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config) {
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
	}

	if err := serveCRMetrics(cfg, operatorNs); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []corev1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*corev1.Service{service}

	// The ServiceMonitor is created in the same namespace where the operator is deployed
	_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, operatorNs string) error {
	// The function below returns a list of filtered operator/CR specific GVKs. For more control, override the GVK list below
	// with your own custom logic. Note that if you are adding third party API schemas, probably you will need to
	// customize this implementation to avoid permissions issues.

	allGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

	// FIXME: Work around https://github.com/operator-framework/operator-sdk/issues/1858
	var ownGVKs []schema.GroupVersionKind
	for _, gvk := range allGVK {
		if !isKubeMetaKind(gvk.Group) {
			continue
		}
		ownGVKs = append(ownGVKs, gvk)
	}
	// The metrics will be generated from the namespaces which are returned here.
	// NOTE that passing nil or an empty list of namespaces in GenerateAndServeCRMetrics will result in an error.
	ns, err := kubemetrics.GetNamespacesForMetrics(operatorNs)
	if err != nil {
		return err
	}

	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, ownGVKs, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

func setupWebhooks(mgr manager.Manager) error {
	rhmiConfigRegister, err := webhooks.WebhookRegisterFor(&integreatlyv1alpha1.RHMIConfig{})
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
				Handler: integreatlyv1alpha1.NewRHMIConfigMutatingHandler(),
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
				Handler: addon.NewDeleteRHMIHandler(mgr.GetConfig()),
			},
		},
	})

	if err := webhooks.Config.SetupServer(mgr); err != nil {
		return err
	}

	return nil
}

func isKubeMetaKind(kind string) bool {
	if strings.HasSuffix(kind, "List") ||
		kind == "PatchOptions" ||
		kind == "GetOptions" ||
		kind == "DeleteOptions" ||
		kind == "ExportOptions" ||
		kind == "APIVersions" ||
		kind == "APIGroupList" ||
		kind == "APIResourceList" ||
		kind == "UpdateOptions" ||
		kind == "CreateOptions" ||
		kind == "Status" ||
		kind == "WatchEvent" ||
		kind == "ListOptions" ||
		kind == "APIGroup" {
		return true
	}

	return false
}
