package openshift

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	postgresProviderName = "openshift-postgres-template"

	defaultPostgresPort        = 5432
	defaultPostgresUser        = "user"
	defaultPostgresPassword    = "password"
	defaultPostgresUserKey     = "user"
	defaultPostgresPasswordKey = "password"
	defaultPostgresDatabaseKey = "database"
	defaultCredentialsSec      = "postgres-credentials"
)

// PostgresStrat to be used to unmarshal strat map
type PostgresStrat struct {
	_ struct{} `type:"structure"`

	PostgresDeploymentSpec *appsv1.DeploymentSpec        `json:"deploymentSpec"`
	PostgresServiceSpec    *v1.ServiceSpec               `json:"serviceSpec"`
	PostgresPVCSpec        *v1.PersistentVolumeClaimSpec `json:"pvcSpec"`
	PostgresSecretData     map[string]string             `json:"secretData"`
}

type OpenShiftPostgresProvider struct {
	Client        client.Client
	Logger        *logrus.Entry
	ConfigManager ConfigManager
}

func NewOpenShiftPostgresProvider(client client.Client, logger *logrus.Entry) *OpenShiftPostgresProvider {
	return &OpenShiftPostgresProvider{
		Client:        client,
		Logger:        logger.WithFields(logrus.Fields{"provider": postgresProviderName}),
		ConfigManager: NewDefaultConfigManager(client),
	}
}

func (p *OpenShiftPostgresProvider) GetName() string {
	return postgresProviderName
}

func (p *OpenShiftPostgresProvider) SupportsStrategy(d string) bool {
	return d == providers.OpenShiftDeploymentStrategy
}

func (p *OpenShiftPostgresProvider) CreatePostgres(ctx context.Context, ps *v1alpha1.Postgres) (*providers.PostgresInstance, v1alpha1.StatusMessage, error) {
	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, ps, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// get postgres config
	postgresCfg, _, err := p.getPostgresConfig(ctx, ps)
	if err != nil {
		return nil, "failed to retrieve openshift postgres config", errorUtil.Wrapf(err, "failed to retrieve openshift postgres config for instance %s", ps.Name)
	}

	// deploy pvc
	if err := p.CreatePVC(ctx, buildDefaultPostgresPVC(ps), postgresCfg); err != nil {
		return nil, "failed to create or update postgres PVC", errorUtil.Wrap(err, "failed to create or update postgres PVC")
	}
	// deploy credentials secret
	if err := p.CreateSecret(ctx, buildDefaultPostgresSecret(ps), postgresCfg); err != nil {
		return nil, "failed to create or update postgres secret", errorUtil.Wrap(err, "failed to create or update postgres secret")
	}
	// deploy deployment
	if err := p.CreateDeployment(ctx, buildDefaultPostgresDeployment(ps), postgresCfg); err != nil {
		return nil, "failed to create or update postgres deployment", errorUtil.Wrap(err, "failed to create or update postgres deployment")
	}
	// deploy service
	if err := p.CreateService(ctx, buildDefaultPostgresService(ps), postgresCfg); err != nil {
		return nil, "failed to create or update postgres service", errorUtil.Wrap(err, "failed to create or update postgres service")
	}

	// check deployment status
	dpl := &appsv1.Deployment{}
	err = p.Client.Get(ctx, types.NamespacedName{Name: ps.Name, Namespace: ps.Namespace}, dpl)
	if err != nil {
		return nil, "failed to get postgres deployment", errorUtil.Wrap(err, "failed to get postgres deployment")
	}

	// get the cred secret
	sec := &v1.Secret{}
	err = p.Client.Get(ctx, types.NamespacedName{Name: defaultCredentialsSec, Namespace: ps.Namespace}, sec)
	if err != nil {
		return nil, "failed to get postgres creds", errorUtil.Wrap(err, "failed to get postgres creds")
	}

	// check if deployment is ready and return connection details
	for _, s := range dpl.Status.Conditions {
		if s.Type == appsv1.DeploymentAvailable && s.Status == "True" {
			p.Logger.Info("Found postgres deployment")
			return &providers.PostgresInstance{
				DeploymentDetails: &providers.PostgresDeploymentDetails{
					Username: string(sec.Data["user"]),
					Password: string(sec.Data["password"]),
					Database: string(sec.Data["database"]),
					Host:     fmt.Sprintf("%s.%s.svc.cluster.local", ps.Name, ps.Namespace),
					Port:     defaultPostgresPort,
				},
			}, "postgres deployment is complete", nil
		}
	}

	// deployment is in progress
	p.Logger.Info("Postgres deployment is not ready")
	return nil, "postgres resources are reconciling", nil
}

func (p *OpenShiftPostgresProvider) DeletePostgres(ctx context.Context, ps *v1alpha1.Postgres) (v1alpha1.StatusMessage, error) {
	// check deployment status
	dpl := &appsv1.Deployment{}
	err := p.Client.Get(ctx, types.NamespacedName{Name: ps.Name, Namespace: ps.Namespace}, dpl)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return "deletion successful", nil
		}
		msg := "failed to get postgres deployment"
		return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	for _, s := range dpl.Status.Conditions {
		if s.Type == appsv1.DeploymentAvailable && s.Status == "True" {
			// delete service
			p.Logger.Info("Deleting postgres service")
			svc := &v1.Service{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      ps.Name,
					Namespace: ps.Namespace,
				},
			}
			err = p.Client.Delete(ctx, svc)
			if err != nil && !k8serr.IsNotFound(err) {
				msg := "failed to delete postgres service"
				return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			// delete pv
			p.Logger.Info("Deleting postgres persistent volume")
			pv := &v1.PersistentVolume{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      ps.Name,
					Namespace: ps.Namespace,
				},
			}
			err = p.Client.Delete(ctx, pv)
			if err != nil && !k8serr.IsNotFound(err) {
				msg := "failed to delete postgres persistent volume"
				return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			// delete pvc
			p.Logger.Info("Deleting postgres persistent volume claim")
			pvc := &v1.PersistentVolumeClaim{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      ps.Name,
					Namespace: ps.Namespace,
				},
			}
			err = p.Client.Delete(ctx, pvc)
			if err != nil && !k8serr.IsNotFound(err) {
				msg := "failed to delete postgres persistent volume claim"
				return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			// delete secret
			p.Logger.Info("Deleting postgres secret")
			sec := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultCredentialsSec,
					Namespace: ps.Namespace,
				},
			}
			err = p.Client.Delete(ctx, sec)
			if err != nil && !k8serr.IsNotFound(err) {
				msg := "failed to deleted postgres secrets"
				return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			// clean up objects
			p.Logger.Info("Deleting postgres deployment")
			err = p.Client.Delete(ctx, dpl)
			if err != nil && !k8serr.IsNotFound(err) {
				msg := "failed to delete postgres deployment"
				return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			// remove the finalizer added by the provider
			p.Logger.Info("Removing postgres finalizer")
			resources.RemoveFinalizer(&ps.ObjectMeta, DefaultFinalizer)
			if err := p.Client.Update(ctx, ps); err != nil {
				msg := "failed to update instance as part of the postgres finalizer reconcile"
				return v1alpha1.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			p.Logger.Infof("deletion handler for postgres %s in namespace %s finished successfully", ps.Name, ps.Namespace)
		}
	}

	return "deletion in progress", nil
}

// getPostgresConfig retrieves the postgres config from the cloud-resources-openshift-strategies configmap
func (p *OpenShiftPostgresProvider) getPostgresConfig(ctx context.Context, ps *v1alpha1.Postgres) (*PostgresStrat, *StrategyConfig, error) {
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, ps.Spec.Tier)
	if err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to read openshift strategy config")
	}

	// unmarshal the postgres config
	postgresCfg := &PostgresStrat{}
	if err := json.Unmarshal(stratCfg.RawStrategy, postgresCfg); err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to unmarshal openshift postgres configuration")
	}

	return postgresCfg, stratCfg, nil
}

func (p *OpenShiftPostgresProvider) CreateDeployment(ctx context.Context, d *appsv1.Deployment, postgresCfg *PostgresStrat) error {
	or, err := controllerutil.CreateOrUpdate(ctx, p.Client, d, func(existing runtime.Object) error {
		e := existing.(*appsv1.Deployment)

		if postgresCfg.PostgresDeploymentSpec == nil {
			e.Spec = d.Spec
			return nil
		}

		e.Spec = *postgresCfg.PostgresDeploymentSpec
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update deployment %s, action was %s", d.Name, or)
	}
	return nil
}

func (p *OpenShiftPostgresProvider) CreateService(ctx context.Context, s *v1.Service, postgresCfg *PostgresStrat) error {
	or, err := controllerutil.CreateOrUpdate(ctx, p.Client, s, func(existing runtime.Object) error {
		e := existing.(*v1.Service)

		if postgresCfg.PostgresServiceSpec == nil {
			e.Spec = s.Spec
			return nil
		}

		e.Spec = *postgresCfg.PostgresServiceSpec
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update service %s, action was %s", s.Name, or)
	}
	return nil
}

func (p *OpenShiftPostgresProvider) CreateSecret(ctx context.Context, s *v1.Secret, postgresCfg *PostgresStrat) error {
	or, err := controllerutil.CreateOrUpdate(ctx, p.Client, s, func(existing runtime.Object) error {
		e := existing.(*v1.Secret)

		if postgresCfg.PostgresSecretData == nil {
			e.Data = s.Data
			return nil
		}

		e.StringData = postgresCfg.PostgresSecretData
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update secret %s, action was %s", s.Name, or)
	}
	return nil
}

func (p *OpenShiftPostgresProvider) CreatePVC(ctx context.Context, pvc *v1.PersistentVolumeClaim, postgresCfg *PostgresStrat) error {
	or, err := controllerutil.CreateOrUpdate(ctx, p.Client, pvc, func(existing runtime.Object) error {
		e := existing.(*v1.PersistentVolumeClaim)

		if postgresCfg.PostgresPVCSpec == nil {
			e.Spec = pvc.Spec
			return nil
		}

		e.Spec = *postgresCfg.PostgresPVCSpec
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update persistent volume claim %s, action was %s", pvc.Name, or)
	}
	return nil
}

func buildDefaultPostgresService(ps *v1alpha1.Postgres) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name,
			Namespace: ps.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "postgresql",
					Protocol:   v1.ProtocolTCP,
					Port:       int32(defaultPostgresPort),
					TargetPort: intstr.FromInt(defaultPostgresPort),
				},
			},
			Selector: map[string]string{"deployment": ps.Name},
		},
	}
}

func buildDefaultPostgresPVC(ps *v1alpha1.Postgres) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name,
			Namespace: ps.Namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func buildDefaultPostgresDeployment(ps *v1alpha1.Postgres) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name,
			Namespace: ps.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": ps.Name,
				},
			},
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: ps.Name,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: ps.Name,
								},
							},
						},
					},
					Containers: buildDefaultPostgresPodContainers(ps),
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": ps.Name,
					},
				},
			},
		},
	}
}

func buildDefaultPostgresPodContainers(ps *v1alpha1.Postgres) []v1.Container {
	return []v1.Container{
		{
			Name:  ps.Name,
			Image: "registry.redhat.io/rhscl/postgresql-96-rhel7",
			Ports: []v1.ContainerPort{
				{
					ContainerPort: int32(defaultPostgresPort),
					Protocol:      v1.ProtocolTCP,
				},
			},
			Env: []v1.EnvVar{
				envVarFromSecret("POSTGRESQL_USER", defaultCredentialsSec, defaultPostgresUserKey),
				envVarFromSecret("POSTGRESQL_PASSWORD", defaultCredentialsSec, defaultPostgresPasswordKey),
				envVarFromSecret("POSTGRESQL_DATABASE", defaultCredentialsSec, defaultPostgresDatabaseKey),
			},
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("250m"),
					v1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("50m"),
					v1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      ps.Name,
					MountPath: "/var/lib/pgsql/data",
				},
			},
			LivenessProbe: &v1.Probe{
				Handler: v1.Handler{
					TCPSocket: &v1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: int32(defaultPostgresPort),
						},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       10,
				TimeoutSeconds:      0,
				SuccessThreshold:    0,
				FailureThreshold:    0,
			},
			ReadinessProbe: &v1.Probe{
				Handler: v1.Handler{
					Exec: &v1.ExecAction{
						Command: []string{"/bin/sh", "-i", "-c", "psql -h 127.0.0.1 -U $POSTGRESQL_USER -q -d $POSTGRESQL_DATABASE -c 'SELECT 1'"}},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       30,
				TimeoutSeconds:      5,
				SuccessThreshold:    0,
				FailureThreshold:    0,
			},
			ImagePullPolicy: v1.PullIfNotPresent,
		},
	}
}

func buildDefaultPostgresSecret(ps *v1alpha1.Postgres) *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCredentialsSec,
			Namespace: ps.Namespace,
		},
		StringData: map[string]string{
			"user":     defaultPostgresUser,
			"password": defaultPostgresPassword,
			"database": ps.Name,
		},
		Type: v1.SecretTypeOpaque,
	}
}

// create an environment variable referencing a secret
func envVarFromSecret(envVarName string, secretName, secretKey string) v1.EnvVar {
	return v1.EnvVar{
		Name: envVarName,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: secretKey,
			},
		},
	}
}
