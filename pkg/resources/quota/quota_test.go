package quota

import (
	"context"
	"reflect"
	"testing"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/utils"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	v1 "github.com/openshift/api/apps/v1"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	TWENTYMILLIONQUOTAPARAM      = "200"
	DEVQUOTAPARAM                = "1"
	TWENTYMILLIONQUOTACONFIGNAME = "20M"
	DEVQUOTACONFIGNAME           = "100K"
)

func TestGetQuota(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	pointerToQuota := &Quota{}

	type args struct {
		QuotaId     string
		QuotaConfig *corev1.ConfigMap
		Quota       *Quota
		isUpdated   bool
		client      client.Client
	}
	tests := []struct {
		name     string
		args     args
		want     *Quota
		wantErr  bool
		validate func(*Quota, *testing.T)
	}{
		{
			name: "ensure error on no quotaid found in config on AWS platform",
			args: args{
				QuotaId:     "QUOTA_NOT_PRESENT_QUOTA",
				QuotaConfig: getQuotaConfig(nil),
				Quota:       pointerToQuota,
				client:      fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(buildTestInfra(configv1.AWSPlatformType)).Build(),
			},
			wantErr: true,
		},
		{
			name: "test successful parsing of config map to quota object for 1 million quota on AWS",
			args: args{
				QuotaId:     DEVQUOTAPARAM,
				QuotaConfig: getQuotaConfig(nil),
				Quota:       pointerToQuota,
				isUpdated:   false,
				client:      fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(buildTestInfra(configv1.AWSPlatformType)).Build(),
			},
			want: &Quota{
				name: DEVQUOTACONFIGNAME,
				productConfigs: map[v1alpha1.ProductName]QuotaProductConfig{
					v1alpha1.Product3Scale: {
						productName: v1alpha1.Product3Scale,
						resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[ApicastProductionName] = ResourceConfig{
								Replicas: int32(1),
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("50m"),
										corev1.ResourceMemory: resource.MustParse("50Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("150m"),
										corev1.ResourceMemory: resource.MustParse("100Mi"),
									},
								},
							}
							rcs[ApicastStagingName] = ResourceConfig{0, corev1.ResourceRequirements{}}
							rcs[BackendListenerName] = ResourceConfig{0, corev1.ResourceRequirements{}}
							rcs[BackendWorkerName] = ResourceConfig{0, corev1.ResourceRequirements{}}
						}),
						quota: pointerToQuota,
					},
					v1alpha1.ProductGrafana: {
						v1alpha1.ProductGrafana,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[GrafanaName] = ResourceConfig{0, corev1.ResourceRequirements{}}
						}),
						pointerToQuota,
					},
					v1alpha1.ProductMarin3r: {
						v1alpha1.ProductMarin3r,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[RateLimitName] = ResourceConfig{0, corev1.ResourceRequirements{}}
						}),
						pointerToQuota,
					},
					v1alpha1.ProductRHSSOUser: {
						v1alpha1.ProductRHSSOUser,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[KeycloakName] = ResourceConfig{0, corev1.ResourceRequirements{}}
						}),
						pointerToQuota,
					},
				},
				isUpdated: false,
				rateLimitConfig: marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 1,
				},
			},
			validate: func(quota *Quota, t *testing.T) {
				gotReplicas := quota.GetProduct(v1alpha1.Product3Scale).GetReplicas(ApicastProductionName)
				wantReplicas := int32(1)
				if gotReplicas != wantReplicas {
					t.Errorf("Expected apicast_production replicas to be '%v' but got '%v'", wantReplicas, gotReplicas)
				}
				gotRequestsPerUnit := quota.GetProduct(v1alpha1.Product3Scale).GetRateLimitConfig().RequestsPerUnit
				wantRequestsPerUnit := uint32(1)
				if gotRequestsPerUnit != wantRequestsPerUnit {
					t.Errorf("Expected requests per unti to be '%v' but got '%v'", wantRequestsPerUnit, gotRequestsPerUnit)
				}
			},
		},
		{
			name: "test successful parsing of config map to quota object for TWENTY million Quota on AWS",
			args: args{
				QuotaId:     TWENTYMILLIONQUOTAPARAM,
				QuotaConfig: getQuotaConfig(nil),
				Quota:       pointerToQuota,
				isUpdated:   false,
				client:      fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(buildTestInfra(configv1.AWSPlatformType)).Build(),
			},
			want: &Quota{
				name: TWENTYMILLIONQUOTACONFIGNAME,
				productConfigs: map[v1alpha1.ProductName]QuotaProductConfig{
					v1alpha1.Product3Scale: {
						productName: v1alpha1.Product3Scale,
						resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[BackendListenerName] = ResourceConfig{
								Replicas: int32(3),
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("0.25"),
										corev1.ResourceMemory: resource.MustParse("450"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("0.3"),
										corev1.ResourceMemory: resource.MustParse("500"),
									},
								},
							}
							rcs[ApicastStagingName] = ResourceConfig{0, corev1.ResourceRequirements{}}
							rcs[ApicastProductionName] = ResourceConfig{0, corev1.ResourceRequirements{}}
							rcs[BackendWorkerName] = ResourceConfig{0, corev1.ResourceRequirements{}}
						}),
						quota: pointerToQuota,
					},
					v1alpha1.ProductGrafana: {
						v1alpha1.ProductGrafana,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[GrafanaName] = ResourceConfig{0, corev1.ResourceRequirements{}}
						}),
						pointerToQuota,
					},
					v1alpha1.ProductMarin3r: {
						productName: v1alpha1.ProductMarin3r,
						resourceConfigs: map[string]ResourceConfig{
							RateLimitName: {0, corev1.ResourceRequirements{}},
						},
						quota: pointerToQuota,
					},
					v1alpha1.ProductRHSSOUser: {
						productName: v1alpha1.ProductRHSSOUser,
						resourceConfigs: map[string]ResourceConfig{
							KeycloakName: {0, corev1.ResourceRequirements{}},
						},
						quota: pointerToQuota,
					},
				},
				isUpdated: false,
				rateLimitConfig: marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 347,
				},
			},
			wantErr: false,
			validate: func(quota *Quota, t *testing.T) {
				gotReplicas := quota.GetProduct(v1alpha1.Product3Scale).GetReplicas(BackendListenerName)
				wantReplicas := int32(3)
				if gotReplicas != wantReplicas {
					t.Errorf("Expected apicast_production replicas to be '%v' but got '%v'", wantReplicas, gotReplicas)
				}
				gotRequestsPerUnit := quota.GetProduct(v1alpha1.Product3Scale).GetRateLimitConfig().RequestsPerUnit
				wantRequestsPerUnit := uint32(347)
				if gotRequestsPerUnit != wantRequestsPerUnit {
					t.Errorf("Expected requests per unti to be '%v' but got '%v'", wantRequestsPerUnit, gotRequestsPerUnit)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GetQuota(context.TODO(), tt.args.client, tt.args.QuotaId, tt.args.QuotaConfig, tt.args.Quota)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetQuota() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil && !reflect.DeepEqual(tt.want, tt.args.Quota) {
				t.Errorf("they don't match, \n got = %v, \n want= %v ", tt.args.Quota, tt.want)
			}

			if tt.validate != nil {
				tt.validate(tt.args.Quota, t)
			}
		})
	}
}

func TestProductConfig_Configure(t *testing.T) {
	type fields struct {
		productName     v1alpha1.ProductName
		resourceConfigs map[string]ResourceConfig
		quota           *Quota
	}
	type args struct {
		obj metav1.Object
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		validate func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T)
		wantErr  bool
	}{
		{
			name: "validate that keycloak rhssouser Resource Requests and Limits get updated",
			fields: fields{
				productName: v1alpha1.ProductRHSSOUser,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[KeycloakName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: getKeycloak(KeycloakName, func(kc *keycloak.Keycloak) {
				kc.Spec.Instances = 2
				kc.Spec.KeycloakDeploymentSpec.Resources = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.111"),
						corev1.ResourceMemory: resource.MustParse("100"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.111"),
						corev1.ResourceMemory: resource.MustParse("100"),
					},
				}
			}),
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				kcSpec := obj.(*keycloak.Keycloak).Spec
				kcLimits := kcSpec.KeycloakDeploymentSpec.Resources.Limits
				configLimits := r[KeycloakName].Resources.Limits
				if kcLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("keycloak cpu limits not as expected, \n got = %v, \n want= %v ", kcLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if kcLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("keycloak memory limits not as expected, \n got = %v, \n want= %v ", kcLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				kcRequests := kcSpec.KeycloakDeploymentSpec.Resources.Requests
				configRequests := r[KeycloakName].Resources.Requests
				if kcRequests.Cpu().MilliValue() != configRequests.Cpu().MilliValue() {
					t.Errorf("keycloak cpu requests not as expected, \n got = %v, \n want= %v ", kcRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if kcRequests.Memory().MilliValue() != configRequests.Memory().MilliValue() {
					t.Errorf("keycloak memory requests not as expected, \n got = %v, \n want= %v ", kcRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				kcReplicas := kcSpec.Instances
				configReplicas := r[KeycloakName].Replicas
				if int32(kcReplicas) != configReplicas {
					t.Errorf("deploymentConfig replicas not as expected, \n got = %v, \n want= %v ", kcReplicas, configReplicas)
				}
			},
			wantErr: false,
		},
		{
			name: "validate that keycloak rhssouser Resource Requests and Limits get updated on install where keycloak resource block is empty",
			fields: fields{
				productName: v1alpha1.ProductRHSSOUser,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[KeycloakName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: getKeycloak(KeycloakName, func(kc *keycloak.Keycloak) {})},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				kcSpec := obj.(*keycloak.Keycloak).Spec
				kcLimits := kcSpec.KeycloakDeploymentSpec.Resources.Limits
				configLimits := r[KeycloakName].Resources.Limits
				if kcLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("keycloak cpu limits not as expected, \n got = %v, \n want= %v ", kcLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if kcLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("keycloak memory limits not as expected, \n got = %v, \n want= %v ", kcLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				kcRequests := kcSpec.KeycloakDeploymentSpec.Resources.Requests
				configRequests := r[KeycloakName].Resources.Requests
				if kcRequests.Cpu().MilliValue() != configRequests.Cpu().MilliValue() {
					t.Errorf("keycloak cpu requests not as expected, \n got = %v, \n want= %v ", kcRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if kcRequests.Memory().MilliValue() != configRequests.Memory().MilliValue() {
					t.Errorf("keycloak memory requests not as expected, \n got = %v, \n want= %v ", kcRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				kcReplicas := kcSpec.Instances
				configReplicas := r[KeycloakName].Replicas
				if int32(kcReplicas) != configReplicas {
					t.Errorf("keycloak replicas not as expected, \n got = %v, \n want= %v ", kcReplicas, configReplicas)
				}
			},
			wantErr: false,
		},
		{
			name: "validate that deploymentConfig backend-listener Resource Requests and Limits get updated",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: getDeploymentConfig(BackendListenerName, func(dc *v1.DeploymentConfig) {
				dc.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.111"),
								corev1.ResourceMemory: resource.MustParse("100"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.111"),
								corev1.ResourceMemory: resource.MustParse("100"),
							},
						},
					},
				}
			},
			),
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				dcSpec := obj.(*v1.DeploymentConfig).Spec
				dcLimits := dcSpec.Template.Spec.Containers[0].Resources.Limits
				configLimits := r[BackendListenerName].Resources.Limits
				if dcLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("deploymentConfig cpu limits not as expected, \n got = %v, \n want= %v ", dcLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if dcLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("deploymentConfig memory limits not as expected, \n got = %v, \n want= %v ", dcLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				dcRequests := dcSpec.Template.Spec.Containers[0].Resources.Requests
				configRequests := r[BackendListenerName].Resources.Requests
				if dcRequests.Cpu().MilliValue() != configRequests.Cpu().MilliValue() {
					t.Errorf("deploymentConfig cpu requests not as expected, \n got = %v, \n want= %v ", dcRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if dcRequests.Memory().MilliValue() != configRequests.Memory().MilliValue() {
					t.Errorf("deploymentConfig memory requests not as expected, \n got = %v, \n want= %v ", dcRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				dcReplicas := dcSpec.Replicas
				configReplicas := r[BackendListenerName].Replicas
				if dcReplicas != configReplicas {
					t.Errorf("deploymentConfig replicas not as expected, \n got = %v, \n want= %v ", dcReplicas, configReplicas)
				}
			},
			wantErr: false,
		},
		{
			name: "validate APIManager case works when resources are nil in apimanager",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[ApicastProductionName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: &threescalev1.APIManager{}},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				apimSpec := obj.(*threescalev1.APIManager).Spec
				resourcesLimits := apimSpec.Backend.ListenerSpec.Resources.Limits
				configLimits := r[BackendListenerName].Resources.Limits
				if resourcesLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("backend listener limits not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if resourcesLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("backend listener limits not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				resourcesRequests := apimSpec.Backend.ListenerSpec.Resources.Requests
				if resourcesRequests.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("backend listener requests not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesRequests.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if resourcesRequests.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("backend listener requests not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesRequests.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}

				//apimSpec.Backend.WorkerSpec.Resources
				resourcesLimits = apimSpec.Backend.WorkerSpec.Resources.Limits
				configLimits = r[BackendWorkerName].Resources.Limits
				if resourcesLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("backend worker limits not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if resourcesLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("backend worker limits not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				resourcesRequests = apimSpec.Backend.WorkerSpec.Resources.Requests
				if resourcesRequests.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("backend worker requests not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesRequests.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if resourcesRequests.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("backend worker requests not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesRequests.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}

				//apimSpec.Apicast.ProductionSpec.Resources
				resourcesLimits = apimSpec.Apicast.ProductionSpec.Resources.Limits
				configLimits = r[ApicastProductionName].Resources.Limits
				if resourcesLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("apicast production limits not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if resourcesLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("apicast production limits not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				resourcesRequests = apimSpec.Apicast.ProductionSpec.Resources.Requests
				configLimits = r[ApicastProductionName].Resources.Requests
				if resourcesRequests.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("apicast production requests not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesRequests.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if resourcesRequests.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("apicast production requests not as expected, values are lower so should update, \n got = %v, \n want= %v ", resourcesRequests.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
			},
			wantErr: false,
		},
		{
			name: "validate that 3scale apimanager Resource Requests and Limits do get updated",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
					rcs[BackendWorkerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
					rcs[ApicastStagingName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
					rcs[ApicastProductionName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: &threescalev1.APIManager{
				Spec: threescalev1.APIManagerSpec{
					Apicast: &threescalev1.ApicastSpec{
						ProductionSpec: &threescalev1.ApicastProductionSpec{
							Resources: nil,
							Replicas:  nil,
						},
						StagingSpec: &threescalev1.ApicastStagingSpec{
							Resources: nil,
							Replicas:  nil,
						},
					},
					Backend: &threescalev1.BackendSpec{
						ListenerSpec: &threescalev1.BackendListenerSpec{
							Resources: nil,
							Replicas:  nil,
						},
						WorkerSpec: &threescalev1.BackendWorkerSpec{
							Resources: nil,
							Replicas:  nil,
						},
					},
				},
			},
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				_ = obj.(*threescalev1.APIManager).Spec
			},
		},
		{
			name: "validate that deploymentConfig backend-listener Resource Requests and Limits do get updated on isUpdated false when values are lower than config",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: false,
				},
			},
			args: args{obj: getDeploymentConfig(BackendListenerName, func(dc *v1.DeploymentConfig) {
				dc.Spec.Replicas = int32(2)
				dc.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.111"),
								corev1.ResourceMemory: resource.MustParse("100"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.111"),
								corev1.ResourceMemory: resource.MustParse("100"),
							},
						},
					},
				}
			}),
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				dcSpec := obj.(*v1.DeploymentConfig).Spec
				dcLimits := dcSpec.Template.Spec.Containers[0].Resources.Limits
				configLimits := r[BackendListenerName].Resources.Limits
				if dcLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("deployment config cpu limits not as expected, values are lower so should update even when isupdated is false, \n got = %v, \n want= %v ", dcLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if dcLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("deployment config memory limits not as expected, values are lower so should update even when isupdated is false, \n got = %v, \n want= %v ", dcLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				dcRequests := dcSpec.Template.Spec.Containers[0].Resources.Requests
				configRequests := r[BackendListenerName].Resources.Requests
				if dcRequests.Cpu().MilliValue() != configRequests.Cpu().MilliValue() {
					t.Errorf("deployment config cpu requests not as expected, values are lower so should update even when isupdated is false, \n got = %v, \n want= %v ", dcRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if dcRequests.Memory().MilliValue() != configRequests.Memory().MilliValue() {
					t.Errorf("deployment config memory requests not as expected, values are lower so should update even when isupdated is false, \n got = %v, \n want= %v ", dcRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				dcReplicas := dcSpec.Replicas
				configReplicas := r[BackendListenerName].Replicas
				if dcReplicas != configReplicas {
					t.Errorf("deploymentConfig replicas not as expected, values are lower so should update even when isupdated is false \n got = %v, \n want= %v ", dcReplicas, configReplicas)
				}
			},
			wantErr: false,
		},
		{
			name: "validate that deploymentConfig backend-listener Resource Requests and Limits do get updated if they are higher and isupdated is true",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.24"),
								corev1.ResourceMemory: resource.MustParse("449"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.24"),
								corev1.ResourceMemory: resource.MustParse("499"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: getDeploymentConfig(BackendListenerName, func(dc *v1.DeploymentConfig) {
				dc.Spec.Replicas = int32(2)
				dc.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					},
				}
			}),
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				dcSpec := obj.(*v1.DeploymentConfig).Spec
				dcLimits := dcSpec.Template.Spec.Containers[0].Resources.Limits
				configLimits := r[BackendListenerName].Resources.Limits
				if dcLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("deployment config cpu limits not as expected, it should update when higher and isupdated is true, \n got = %v, \n want= %v ", dcLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if dcLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("deployment config memory limits not as expected, it should update when higher and isupdated is true, \n got = %v, \n want= %v ", dcLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				dcRequests := dcSpec.Template.Spec.Containers[0].Resources.Requests
				configRequests := r[BackendListenerName].Resources.Requests
				if dcRequests.Cpu().MilliValue() != configRequests.Cpu().MilliValue() {
					t.Errorf("deployment config cpu requests not as expected, it should update when higher and isupdated is true, \n got = %v, \n want= %v ", dcRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if dcRequests.Memory().MilliValue() != configRequests.Memory().MilliValue() {
					t.Errorf("deployment config memory requests not as expected, it should update when higher and isupdated is true, \n got = %v, \n want= %v ", dcRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				dcReplicas := dcSpec.Replicas
				configReplicas := r[BackendListenerName].Replicas
				if dcReplicas != configReplicas {
					t.Errorf("deploymentConfig replicas not as expected, \n got = %v, \n want= %v ", dcReplicas, configReplicas)
				}
			},
			wantErr: false,
		},
		{
			name: "validate that statefulset backend-listener Resource Requests and Limits do get updated if they are higher and isupdated is true",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.24"),
								corev1.ResourceMemory: resource.MustParse("449"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.24"),
								corev1.ResourceMemory: resource.MustParse("499"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: true,
				},
			},
			args: args{obj: getStatefulSet(BackendListenerName, func(ss *appsv1.StatefulSet) {
				replica := int32(2)
				ss.Spec.Replicas = &replica
				ss.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					},
				}
			}),
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				ssSpec := obj.(*appsv1.StatefulSet).Spec
				ssLimits := ssSpec.Template.Spec.Containers[0].Resources.Limits
				configLimits := r[BackendListenerName].Resources.Limits
				if ssLimits.Cpu().MilliValue() != configLimits.Cpu().MilliValue() {
					t.Errorf("statefulset cpu limits not as expected, it should get update when lower, \n got = %v, \n want= %v ", ssLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if ssLimits.Memory().MilliValue() != configLimits.Memory().MilliValue() {
					t.Errorf("statefulset memory limits not as expected, it should get update when lower, \n got = %v, \n want= %v ", ssLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				ssRequests := ssSpec.Template.Spec.Containers[0].Resources.Requests
				configRequests := r[BackendListenerName].Resources.Requests
				if ssRequests.Cpu().MilliValue() != configRequests.Cpu().MilliValue() {
					t.Errorf("statefulset cpu requests not as expected, it should get update when lower, \n got = %v, \n want= %v ", ssRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if ssRequests.Memory().MilliValue() != configRequests.Memory().MilliValue() {
					t.Errorf("statefulset memory requests not as expected, it should get update when lower, \n got = %v, \n want= %v ", ssRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				ssReplicas := ssSpec.Replicas
				configReplicas := r[BackendListenerName].Replicas
				if *ssReplicas != configReplicas {
					t.Errorf("statefulset replicas not as expected, \n got = %v, \n want= %v ", *ssReplicas, configReplicas)
				}
			},
			wantErr: false,
		},
		{
			name: "validate that deployment backend-listener Resource Requests and Limits don't get updated if they are higher and isupdated is false",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.24"),
								corev1.ResourceMemory: resource.MustParse("449"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.24"),
								corev1.ResourceMemory: resource.MustParse("499"),
							},
						},
					}
				}),
				quota: &Quota{
					isUpdated: false,
				},
			},
			args: args{obj: getDeployment(BackendListenerName, func(d *appsv1.Deployment) {
				d.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.25"),
								corev1.ResourceMemory: resource.MustParse("450"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("0.3"),
								corev1.ResourceMemory: resource.MustParse("500"),
							},
						},
					},
				}
			}),
			},
			validate: func(obj metav1.Object, r map[string]ResourceConfig, t *testing.T) {
				dSpec := obj.(*appsv1.Deployment).Spec
				dLimits := dSpec.Template.Spec.Containers[0].Resources.Limits
				configLimits := r[BackendListenerName].Resources.Limits
				if dLimits.Cpu().MilliValue() == configLimits.Cpu().MilliValue() {
					t.Errorf("deployment cpu limits not as expected, it should not update when lower and isUpdate is false, \n got = %v, \n want= %v ", dLimits.Cpu().MilliValue(), configLimits.Cpu().MilliValue())
				}
				if dLimits.Memory().MilliValue() == configLimits.Memory().MilliValue() {
					t.Errorf("deployment memory limits not as expected, it should not update when lower and isUpdate is false, \n got = %v, \n want= %v ", dLimits.Memory().MilliValue(), configLimits.Memory().MilliValue())
				}
				dRequests := dSpec.Template.Spec.Containers[0].Resources.Requests
				configRequests := r[BackendListenerName].Resources.Requests
				if dRequests.Cpu().MilliValue() == configRequests.Cpu().MilliValue() {
					t.Errorf("deployment cpu requests not as expected, it should not update when lower and isUpdate is false, \n got = %v, \n want= %v ", dRequests.Cpu().MilliValue(), configRequests.Cpu().MilliValue())
				}
				if dRequests.Memory().MilliValue() == configRequests.Memory().MilliValue() {
					t.Errorf("deployment memory requests not as expected, it should not update when lower and isUpdate is false, \n got = %v, \n want= %v ", dRequests.Memory().MilliValue(), configRequests.Memory().MilliValue())
				}
				dReplicas := dSpec.Replicas
				configReplicas := r[BackendListenerName].Replicas
				if *dReplicas != configReplicas {
					t.Errorf("deployment replicas not as expected, \n got = %v, \n want= %v ", *dReplicas, configReplicas)
				}
			},
		},
		{
			name: "validate error returned on non deployment deploymentConfig or StatefulSet Object passed",
			args: args{obj: &corev1.ConfigMap{}},
			fields: fields{
				quota: &Quota{
					isUpdated: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &QuotaProductConfig{
				productName:     tt.fields.productName,
				resourceConfigs: tt.fields.resourceConfigs,
				quota:           tt.fields.quota,
			}
			err := p.Configure(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(tt.args.obj, tt.fields.resourceConfigs, t)
			}
		})
	}
}

func getResourceConfig(modifyFn func(rcs map[string]ResourceConfig)) map[string]ResourceConfig {
	mock := map[string]ResourceConfig{}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func getKeycloak(name string, modifyFn func(kc *keycloak.Keycloak)) *keycloak.Keycloak {
	mock := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func getDeploymentConfig(name string, modifyFn func(dc *v1.DeploymentConfig)) *v1.DeploymentConfig {
	mock := &v1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.DeploymentConfigSpec{
			Template: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{},
				},
			},
		}}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func getStatefulSet(name string, modifyFn func(ss *appsv1.StatefulSet)) *appsv1.StatefulSet {
	mock := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{},
				},
			},
		}}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func getDeployment(name string, modifyFn func(d *appsv1.Deployment)) *appsv1.Deployment {
	mock := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{},
				},
			},
		}}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func getQuotaConfig(modifyFn func(*corev1.ConfigMap)) *corev1.ConfigMap {
	mock := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName},
	}
	mock.Data = map[string]string{
		ConfigMapData: "[{\"name\": \"" + DEVQUOTACONFIGNAME + "\",\"param\": \"" + DEVQUOTAPARAM + "\",\"rate-limiting\": {\"unit\": \"minute\",\"requests_per_unit\": 1, \"alert_limits\": []},\"resources\": {\"" + ApicastProductionName + "\": {\"replicas\": 1,\"resources\": {\"requests\": {\"cpu\": \"50m\",\"memory\": \"50Mi\"},\"limits\": {\"cpu\": \"150m\",\"memory\": \"100Mi\"}}}}}, {\"name\": \"" + TWENTYMILLIONQUOTACONFIGNAME + "\",\"param\": \"" + TWENTYMILLIONQUOTAPARAM + "\",\"rate-limiting\": {  \"unit\": \"minute\",  \"requests_per_unit\": 347,  \"alert_limits\": []},\"resources\": {\"" + BackendListenerName + "\": {\"replicas\": 3,\"resources\": {  \"requests\": {\"cpu\": 0.25,\"memory\": 450  },  \"limits\": {\"cpu\": 0.3,\"memory\": 500}}}}}]",
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func buildTestInfra(platformType configv1.PlatformType) *configv1.Infrastructure {
	return &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: configv1.InfrastructureStatus{
			PlatformStatus: &configv1.PlatformStatus{
				Type: platformType,
			},
		},
	}
}
