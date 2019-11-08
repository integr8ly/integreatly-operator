package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	types2 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"strings"
	"time"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	errorUtil "github.com/pkg/errors"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	redisProviderName = "openshift-redis-template"
	// default create options
	redisObjectMetaName   = "redis"
	redisDCSelectorName   = redisObjectMetaName
	redisConfigVolumeName = "redis-config"
	redisConfigMapName    = "redis-config"
	redisConfigMapKey     = "redis.conf"
	redisContainerName    = "redis"
	redisPort             = 6379
	redisContainerCommand = "/opt/rh/rh-redis32/root/usr/bin/redis-server"
)

var _ providers.RedisProvider = (*OpenShiftRedisProvider)(nil)

type OpenShiftRedisProvider struct {
	Client        client.Client
	Logger        *logrus.Entry
	ConfigManager ConfigManager
}

func NewOpenShiftRedisProvider(client client.Client, logger *logrus.Entry) *OpenShiftRedisProvider {
	return &OpenShiftRedisProvider{
		Client:        client,
		Logger:        logger.WithFields(logrus.Fields{"provider": redisProviderName}),
		ConfigManager: NewDefaultConfigManager(client),
	}
}

func (p *OpenShiftRedisProvider) GetName() string {
	return redisProviderName
}

func (p *OpenShiftRedisProvider) SupportsStrategy(d string) bool {
	return d == providers.OpenShiftDeploymentStrategy
}

func (p *OpenShiftRedisProvider) GetReconcileTime(r *v1alpha1.Redis) time.Duration {
	if r.Status.Phase != types2.PhaseComplete {
		return time.Second * 10
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (p *OpenShiftRedisProvider) CreateRedis(ctx context.Context, r *v1alpha1.Redis) (*providers.RedisCluster, types2.StatusMessage, error) {
	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, r, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// get redis config
	redisConfig, _, err := p.getRedisConfig(ctx, r)
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve openshift redis cluster config for instance %s", r.Name)
		return nil, types2.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// deploy pvc
	if err := p.CreatePVC(ctx, buildDefaultRedisPVC(r), redisConfig); err != nil {
		errMsg := "failed to create or update redis PVC"
		return nil, types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	// deploy configmap
	if err := p.CreateConfigMap(ctx, buildDefaultRedisConfigMap(r), redisConfig); err != nil {
		errMsg := "failed to create or update redis config map"
		return nil, types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	// deploy deployment
	if err := p.CreateDeployment(ctx, buildDefaultRedisDeployment(r), redisConfig); err != nil {
		errMsg := "failed to create or update redis deployment"
		return nil, types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	// deploy service
	if err := p.CreateService(ctx, buildDefaultRedisService(r), redisConfig); err != nil {
		errMsg := "failed to create or update redis service"
		return nil, types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check deployment status
	dpl := &appsv1.Deployment{}
	err = p.Client.Get(ctx, types.NamespacedName{Name: r.Name, Namespace: r.Namespace}, dpl)
	if err != nil {
		errMsg := "failed to get redis deployment"
		return nil, types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check if deployment is ready and return connection details
	for _, s := range dpl.Status.Conditions {
		if s.Type == appsv1.DeploymentAvailable && s.Status == "True" {
			p.Logger.Info("found redis deployment")
			return &providers.RedisCluster{DeploymentDetails: &providers.RedisDeploymentDetails{
				URI:  fmt.Sprintf("%s.%s.svc.cluster.local", r.Name, r.Namespace),
				Port: redisPort}}, "redis deployment available", nil
		}
	}

	// deployment is in progress
	p.Logger.Info("redis deployment is not ready")
	return nil, "creation in progress", nil
}

func (p *OpenShiftRedisProvider) DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (types2.StatusMessage, error) {
	// check deployment status
	dpl := &appsv1.Deployment{}
	err := p.Client.Get(ctx, types.NamespacedName{Name: r.Name, Namespace: r.Namespace}, dpl)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return "deletion successful", nil
		}
		errMsg := "failed to get deployment"
		return types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	for _, s := range dpl.Status.Conditions {
		if s.Type == appsv1.DeploymentAvailable && s.Status == "True" {
			// delete service
			p.Logger.Info("Deleting redis service")
			svc := &apiv1.Service{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      r.Name,
					Namespace: r.Namespace,
				},
			}
			err = p.Client.Delete(ctx, svc)
			if err != nil && !k8serr.IsNotFound(err) {
				errMsg := "failed to delete service"
				return types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}

			// delete pvc
			p.Logger.Info("Deleting redis persistent volume claim")
			pvc := &apiv1.PersistentVolumeClaim{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      r.Name,
					Namespace: r.Namespace,
				},
			}
			err = p.Client.Delete(ctx, pvc)
			if err != nil && !k8serr.IsNotFound(err) {
				errMsg := "failed to delete persistent volume claim"
				return types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}

			// delete config map
			p.Logger.Info("Deleting redis configmap")
			cm := &apiv1.ConfigMap{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      redisConfigMapName,
					Namespace: r.Namespace,
				},
			}
			err = p.Client.Delete(ctx, cm)
			if err != nil && !k8serr.IsNotFound(err) {
				errMsg := "failed to delete configmap"
				return types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}

			// clean up objects
			p.Logger.Info("Deleting redis deployment")
			err = p.Client.Delete(ctx, dpl)
			if err != nil && !k8serr.IsNotFound(err) {
				errMsg := "failed to delete deployment"
				return types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}

			// remove the finalizer added by the provider
			p.Logger.Info("Removing finalizer")
			resources.RemoveFinalizer(&r.ObjectMeta, DefaultFinalizer)
			if err := p.Client.Update(ctx, r); err != nil {
				errMsg := "failed to update instance as part of finalizer reconcile"
				return types2.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}

			p.Logger.Infof("deletion handler for redis %s in namespace %s finished successfully", r.Name, r.Namespace)
		}
	}

	return "deletion in progress", nil
}

// getPostgresConfig retrieves the redis config from the cloud-resources-openshift-strategies configmap
func (p *OpenShiftRedisProvider) getRedisConfig(ctx context.Context, r *v1alpha1.Redis) (*RedisStrat, *StrategyConfig, error) {
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.RedisResourceType, r.Spec.Tier)
	if err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to read openshift strategy config")
	}

	// unmarshal the redis cluster config
	redisConfig := &RedisStrat{}
	if err := json.Unmarshal(stratCfg.RawStrategy, redisConfig); err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to unmarshal openshift redis cluster configuration")
	}
	return redisConfig, stratCfg, nil
}

func (p *OpenShiftRedisProvider) CreateDeployment(ctx context.Context, d *appsv1.Deployment, redisCfg *RedisStrat) error {
	or, err := immutableCreateOrUpdate(ctx, p.Client, d, func(existing runtime.Object) error {
		e := existing.(*appsv1.Deployment)
		if redisCfg.RedisDeploymentSpec == nil {
			e.Spec = d.Spec
			return nil
		}

		e.Spec = *redisCfg.RedisDeploymentSpec
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update deployment %s, action was %s", d.Name, or)
	}
	return nil
}

func (p *OpenShiftRedisProvider) CreateService(ctx context.Context, s *apiv1.Service, redisCfg *RedisStrat) error {
	or, err := immutableCreateOrUpdate(ctx, p.Client, s, func(existing runtime.Object) error {
		e := existing.(*apiv1.Service)

		if redisCfg.RedisServiceSpec == nil {
			clusterIP := e.Spec.ClusterIP
			e.Spec = s.Spec
			e.Spec.ClusterIP = clusterIP
			return nil
		}

		e.Spec = *redisCfg.RedisServiceSpec
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update service %s, action was %s", s.Name, or)
	}
	return nil
}

func (p *OpenShiftRedisProvider) CreateConfigMap(ctx context.Context, cm *apiv1.ConfigMap, redisCfg *RedisStrat) error {
	or, err := immutableCreateOrUpdate(ctx, p.Client, cm, func(existing runtime.Object) error {
		e := existing.(*apiv1.ConfigMap)

		if redisCfg.RedisConfigMapData == nil {
			e.Data = cm.Data
			return nil
		}

		e.Data = redisCfg.RedisConfigMapData
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update config map %s, action was %s", cm.Name, or)
	}
	return nil
}

func (p *OpenShiftRedisProvider) CreatePVC(ctx context.Context, pvc *apiv1.PersistentVolumeClaim, redisCfg *RedisStrat) error {
	or, err := immutableCreateOrUpdate(ctx, p.Client, pvc, func(existing runtime.Object) error {
		e := existing.(*apiv1.PersistentVolumeClaim)
		// resources.requests is only mutable on bound claims
		if strings.ToLower(string(e.Status.Phase)) != "bound" {
			return nil
		}
		if redisCfg.RedisPVCSpec == nil {
			e.Spec.Resources.Requests = pvc.Spec.Resources.Requests
			return nil
		}
		e.Spec.Resources.Requests = redisCfg.RedisPVCSpec.Resources.Requests
		return nil
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create or update persistent volume claim %s, action was %s", pvc.Name, or)
	}
	return nil
}

// RedisStrat to be used to unmarshal strat map
type RedisStrat struct {
	_ struct{} `type:"structure"`

	RedisDeploymentSpec *appsv1.DeploymentSpec           `json:"deploymentSpec"`
	RedisServiceSpec    *apiv1.ServiceSpec               `json:"serviceSpec"`
	RedisPVCSpec        *apiv1.PersistentVolumeClaimSpec `json:"pvcSpec"`
	RedisConfigMapData  map[string]string                `json:"configMapData"`
}

func buildDefaultRedisDeployment(r *v1alpha1.Redis) *appsv1.Deployment {
	dc := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					Volumes:    buildDefaultRedisPodVolumes(r),
					Containers: buildDefaultRedisPodContainers(r),
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": redisDCSelectorName,
					},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": redisDCSelectorName,
				},
			},
			Replicas: int32Ptr(1),
		},
	}
	return dc
}

func buildDefaultRedisPodContainers(r *v1alpha1.Redis) []apiv1.Container {
	return []apiv1.Container{
		{
			Image:           "registry.redhat.io/rhscl/redis-32-rhel7",
			ImagePullPolicy: apiv1.PullIfNotPresent,
			Name:            redisContainerName,
			Command: []string{
				redisContainerCommand,
			},
			Args: []string{
				"/etc/redis.d/redis.conf",
				"--daemonize",
				"no",
			},
			Resources: apiv1.ResourceRequirements{
				Limits: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("500m"),
					apiv1.ResourceMemory: resource.MustParse("32Gi"),
				},
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("150m"),
					apiv1.ResourceMemory: resource.MustParse("256Mi"),
				},
			},
			ReadinessProbe: &apiv1.Probe{
				Handler: apiv1.Handler{
					Exec: &apiv1.ExecAction{
						Command: []string{
							"container-entrypoint",
							"bash",
							"-c",
							"redis-cli set liveness-probe \"`date`\" | grep OK",
						},
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       30,
				TimeoutSeconds:      1,
			},
			LivenessProbe: &apiv1.Probe{
				InitialDelaySeconds: 10,
				PeriodSeconds:       10,
				Handler: apiv1.Handler{
					TCPSocket: &apiv1.TCPSocketAction{
						Port: intstr.FromInt(6379),
					},
				},
			},
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      r.Name,
					MountPath: "/var/lib/redis/data",
				},
				{
					Name:      redisConfigVolumeName,
					MountPath: "/etc/redis.d/",
				},
			},
		},
	}
}

func buildDefaultRedisPodVolumes(r *v1alpha1.Redis) []apiv1.Volume {
	return []apiv1.Volume{
		{
			Name: r.Name,
			VolumeSource: apiv1.VolumeSource{
				PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
					ClaimName: r.Name,
				},
			},
		},
		{
			Name: redisConfigVolumeName,
			VolumeSource: apiv1.VolumeSource{
				ConfigMap: &apiv1.ConfigMapVolumeSource{
					LocalObjectReference: apiv1.LocalObjectReference{
						Name: redisConfigMapName, // the name of the ConfigMap
					},
					Items: []apiv1.KeyToPath{
						{
							Key:  redisConfigMapKey,
							Path: redisConfigMapKey,
						},
					},
				},
			},
		},
	}
}

func buildDefaultRedisService(r *v1alpha1.Redis) *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
					Protocol:   apiv1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"deployment": redisDCSelectorName,
			},
		},
	}
}

func buildDefaultRedisConfigMap(r *v1alpha1.Redis) *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redisConfigMapName,
			Namespace: r.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		Data: map[string]string{
			"redis.conf": getRedisConfData(),
		},
	}
}

func getRedisConfData() string {
	return `protected-mode no
port 6379
timeout 0
tcp-keepalive 300
daemonize no
supervised no
loglevel notice
databases 16
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
slave-serve-stale-data yes
slave-read-only yes
repl-diskless-sync no
repl-disable-tcp-nodelay no
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
aof-load-truncated yes
lua-time-limit 5000
activerehashing no
aof-rewrite-incremental-fsync yes
dir /var/lib/redis/data
`
}

func buildDefaultRedisPVC(r *v1alpha1.Redis) *apiv1.PersistentVolumeClaim {
	return &apiv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		Spec: apiv1.PersistentVolumeClaimSpec{
			AccessModes: []apiv1.PersistentVolumeAccessMode{
				apiv1.ReadWriteOnce,
			},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func int32Ptr(i int32) *int32 { return &i }

// controllerutil.CreateOrUpdate without mutating the original runtime.Object provided
func immutableCreateOrUpdate(ctx context.Context, c client.Client, o runtime.Object, cb func(existing runtime.Object) error) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrUpdate(ctx, c, o.DeepCopyObject(), cb)
}
