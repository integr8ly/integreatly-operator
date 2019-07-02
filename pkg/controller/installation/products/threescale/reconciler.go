package threescale

import (
	"context"
	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultInstallationNamespace = "3scale"
	packageName                  = "3scale"
	apiManagerName               = "3scale"
)

func NewReconciler(client pkgclient.Client, rc *rest.Config, configManager config.ConfigReadWriter, i *v1alpha1.Installation, mgr manager.Manager) (*Reconciler, error) {
	mpm := marketplace.NewManager(client, mgr, rc)
	return &Reconciler{
		client:        client,
		restConfig:    rc,
		ConfigManager: configManager,
		mpm:           mpm,
		mgr:           mgr,
		namespace:     i.Spec.NamespacePrefix + defaultInstallationNamespace,
	}, nil
}

type Reconciler struct {
	client        pkgclient.Client
	restConfig    *rest.Config
	namespace     string
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	mgr           manager.Manager
}

func (r *Reconciler) Reconcile(in *v1alpha1.Installation) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", packageName)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.namespace,
			Labels: map[string]string{
				"integreatly": "yes",
			},
		},
	}
	err := r.client.Get(context.TODO(), pkgclient.ObjectKey{Name: r.namespace}, ns)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}

	if err != nil {
		logrus.Infof("Namespace %s not present", r.namespace)
		if err := controllerutil.SetControllerReference(in, ns, r.mgr.GetScheme()); err != nil {
			return v1alpha1.PhaseFailed, err
		}
		err := r.client.Create(context.TODO(), ns)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
		logrus.Infof("%s namespace created", r.namespace)
	}

	if ns.Status.Phase == v1.NamespaceActive {
		serverClient, err := pkgclient.New(r.restConfig, pkgclient.Options{})
		if err != nil {
			logrus.Infof("Error creating server client")
			return v1alpha1.PhaseFailed, err
		}

		err = r.mpm.CreateSubscription(
			in,
			marketplace.GetOperatorSources().Integreatly,
			r.namespace,
			packageName,
			"integreatly",
			[]string{r.namespace},
			coreosv1alpha1.ApprovalAutomatic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, err
		}

		ip, err := r.mpm.GetSubscriptionInstallPlan(packageName, r.namespace)
		if ip != nil && ip.Status.Phase == coreosv1alpha1.InstallPlanPhaseComplete {
			logrus.Infof("3scale operator installed")
			s := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s3-credentials",
					Namespace: r.namespace,
					Labels: map[string]string{
						"integreatly": "yes",
					},
				},
				StringData: map[string]string{
					"AWS_ACCESS_KEY_ID":     "DUMMY_ACCESS_KEY",
					"AWS_SECRET_ACCESS_KEY": "DUMMY_SECRET_KEY",
				},
			}

			err = serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: s.Name, Namespace: r.namespace}, s)
			if err != nil && !k8serr.IsNotFound(err) {
				return v1alpha1.PhaseFailed, err
			}

			if err != nil {
				if err := controllerutil.SetControllerReference(in, s, r.mgr.GetScheme()); err != nil {
					return v1alpha1.PhaseFailed, err
				}
				err := serverClient.Create(context.TODO(), s)
				if err != nil {
					return v1alpha1.PhaseFailed, err
				}
			}

			resourceRequirements := false
			apim := &threescalev1.APIManager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      apiManagerName,
					Namespace: r.namespace,
					Labels: map[string]string{
						"integreatly": "yes",
					},
				},
				Spec: threescalev1.APIManagerSpec{
					APIManagerCommonSpec: threescalev1.APIManagerCommonSpec{
						WildcardDomain:              in.Spec.RoutingSubdomain,
						ResourceRequirementsEnabled: &resourceRequirements,
					},
					System: &threescalev1.SystemSpec{
						FileStorageSpec: &threescalev1.SystemFileStorageSpec{
							S3: &threescalev1.SystemS3Spec{
								AWSBucket: "jroche-test",
								AWSRegion: "eu-central-1",
								AWSCredentials: v1.LocalObjectReference{
									Name: "s3-credentials",
								},
							},
						},
					},
				},
			}
			err = serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: apim.Name, Namespace: r.namespace}, apim)
			if err != nil && !k8serr.IsNotFound(err) {
				return v1alpha1.PhaseFailed, err
			}

			if err != nil {
				logrus.Infof("Creating API Manager")
				err := serverClient.Create(context.TODO(), apim)
				if err != nil {
					return v1alpha1.PhaseFailed, err
				}
			} else {
				if len(apim.Status.Deployments.Starting) == 0 && len(apim.Status.Deployments.Stopped) == 0 && len(apim.Status.Deployments.Ready) > 0 {
					logrus.Infof("%s has successfully deployed", packageName)
					return v1alpha1.PhaseCompleted, nil
				}
			}
		}
	}

	return v1alpha1.PhaseInProgress, nil
}
