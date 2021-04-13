package sku

import (
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	v1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

const (
	TWENTYMILLIONSKU = "20"
	DEVSKU           = "1"
)

func TestGetSKU(t *testing.T) {

	pointerToSKU := &SKU{}

	type args struct {
		SKUId     string
		SKUConfig *corev1.ConfigMap
		SKU       *SKU
		isUpdated bool
	}
	tests := []struct {
		name     string
		args     args
		want     *SKU
		wantErr  bool
		validate func(*SKU, *testing.T)
	}{
		{
			name: "ensure error on no skuid found in config",
			args: args{
				SKUId:     "SKU_NOT_PRESENT_SKU",
				SKUConfig: getSKUConfig(nil),
				SKU:       pointerToSKU,
			},
			wantErr: true,
		},
		{
			name: "test successful parsing of config map to sku object for 1 million sku",
			args: args{
				SKUId:     DEVSKU,
				SKUConfig: getSKUConfig(nil),
				SKU:       pointerToSKU,
				isUpdated: false,
			},
			want: &SKU{
				name: DEVSKU,
				productConfigs: map[v1alpha1.ProductName]AProductConfig{
					v1alpha1.Product3Scale: {
						productName: v1alpha1.Product3Scale,
						resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[ApicastProductionName] = ResourceConfig{
								Replicas: int32(1),
								Resources: v13.ResourceRequirements{
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
							rcs[ApicastStagingName] = ResourceConfig{0, v13.ResourceRequirements{}}
							rcs[BackendListenerName] = ResourceConfig{0, v13.ResourceRequirements{}}
							rcs[BackendWorkerName] = ResourceConfig{0, v13.ResourceRequirements{}}
						}),
						sku: pointerToSKU,
					},
					v1alpha1.ProductGrafana: {
						v1alpha1.ProductGrafana,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[GrafanaName] = ResourceConfig{0, v13.ResourceRequirements{}}
						}),
						pointerToSKU,
					},
					v1alpha1.ProductMarin3r: {
						v1alpha1.ProductMarin3r,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[RateLimitName] = ResourceConfig{0, v13.ResourceRequirements{}}
						}),
						pointerToSKU,
					},
					v1alpha1.ProductRHSSOUser: {
						v1alpha1.ProductRHSSOUser,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[KeycloakName] = ResourceConfig{0, v13.ResourceRequirements{}}
						}),
						pointerToSKU,
					},
				},
				isUpdated: false,
				rateLimitConfig: marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 1,
				},
			},
			validate: func(sku *SKU, t *testing.T) {
				gotReplicas := sku.GetProduct(v1alpha1.Product3Scale).GetReplicas(ApicastProductionName)
				wantReplicas := int32(1)
				if gotReplicas != wantReplicas {
					t.Errorf("Expected apicast_production replicas to be '%v' but got '%v'", wantReplicas, gotReplicas)
				}
			},
		},
		{
			name: "test successful parsing of config map to sku object for TWENTY million SKU",
			args: args{
				SKUId:     TWENTYMILLIONSKU,
				SKUConfig: getSKUConfig(nil),
				SKU:       pointerToSKU,
				isUpdated: false,
			},
			want: &SKU{
				name: TWENTYMILLIONSKU,
				productConfigs: map[v1alpha1.ProductName]AProductConfig{
					v1alpha1.Product3Scale: {
						productName: v1alpha1.Product3Scale,
						resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[BackendListenerName] = ResourceConfig{
								Replicas: int32(3),
								Resources: v13.ResourceRequirements{
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
							rcs[ApicastStagingName] = ResourceConfig{0, v13.ResourceRequirements{}}
							rcs[ApicastProductionName] = ResourceConfig{0, v13.ResourceRequirements{}}
							rcs[BackendWorkerName] = ResourceConfig{0, v13.ResourceRequirements{}}
						}),
						sku: pointerToSKU,
					},
					v1alpha1.ProductGrafana: {
						v1alpha1.ProductGrafana,
						getResourceConfig(func(rcs map[string]ResourceConfig) {
							rcs[GrafanaName] = ResourceConfig{0, v13.ResourceRequirements{}}
						}),
						pointerToSKU,
					},
					v1alpha1.ProductMarin3r: {
						productName: v1alpha1.ProductMarin3r,
						resourceConfigs: map[string]ResourceConfig{
							RateLimitName: {0, v13.ResourceRequirements{}},
						},
						sku: pointerToSKU,
					},
					v1alpha1.ProductRHSSOUser: {
						productName: v1alpha1.ProductRHSSOUser,
						resourceConfigs: map[string]ResourceConfig{
							KeycloakName: {0, v13.ResourceRequirements{}},
						},
						sku: pointerToSKU,
					},
				},
				isUpdated: false,
				rateLimitConfig: marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 347,
				},
			},
			wantErr: false,
			validate: func(sku *SKU, t *testing.T) {
				gotReplicas := sku.GetProduct(v1alpha1.Product3Scale).GetReplicas(BackendListenerName)
				wantReplicas := int32(3)
				if gotReplicas != wantReplicas {
					t.Errorf("Expected apicast_production replicas to be '%v' but got '%v'", wantReplicas, gotReplicas)
				}
				// to do more interrogations
				// check the rate limit amounts
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GetSKU(tt.args.SKUId, tt.args.SKUConfig, tt.args.SKU, tt.args.isUpdated)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSKU() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want != nil && !reflect.DeepEqual(tt.want, tt.args.SKU) {
				t.Errorf("they don't match, \n got = %v, \n want= %v ", tt.args.SKU, tt.want)
			}

			if tt.validate != nil {
				tt.validate(tt.args.SKU, t)
			}
		})
	}
}

func TestProductConfig_Configure(t *testing.T) {
	type fields struct {
		productName     v1alpha1.ProductName
		resourceConfigs map[string]ResourceConfig
		sku             *SKU
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
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
					isUpdated: true,
				},
			},
			args: args{obj: getKeycloak(KeycloakName, func(kc *keycloak.Keycloak) {
				kc.Spec.Instances = 2
				kc.Spec.KeycloakDeploymentSpec.Resources = v13.ResourceRequirements{
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
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
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
					t.Errorf("deploymentConfig replicas not as expected, \n got = %v, \n want= %v ", kcReplicas, configReplicas)
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
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
					isUpdated: true,
				},
			},
			args: args{obj: getDeploymentConfig(BackendListenerName, func(dc *v1.DeploymentConfig) {
				dc.Spec.Replicas = int32(2)
				dc.Spec.Template.Spec.Containers = []v13.Container{
					{
						Resources: v13.ResourceRequirements{
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
			name: "validate that deploymentConfig backend-listener Resource Requests and Limits do get updated on isUpdated false when values are lower than config",
			fields: fields{
				productName: v1alpha1.Product3Scale,
				resourceConfigs: getResourceConfig(func(rcs map[string]ResourceConfig) {
					rcs[BackendListenerName] = ResourceConfig{
						Replicas: int32(3),
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
					isUpdated: false,
				},
			},
			args: args{obj: getDeploymentConfig(BackendListenerName, func(dc *v1.DeploymentConfig) {
				dc.Spec.Replicas = int32(2)
				dc.Spec.Template.Spec.Containers = []v13.Container{
					{
						Resources: v13.ResourceRequirements{
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
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
					isUpdated: true,
				},
			},
			args: args{obj: getDeploymentConfig(BackendListenerName, func(dc *v1.DeploymentConfig) {
				dc.Spec.Replicas = int32(2)
				dc.Spec.Template.Spec.Containers = []v13.Container{
					{
						Resources: v13.ResourceRequirements{
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
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
					isUpdated: true,
				},
			},
			args: args{obj: getStatefulSet(BackendListenerName, func(ss *appsv1.StatefulSet) {
				replica := int32(2)
				ss.Spec.Replicas = &replica
				ss.Spec.Template.Spec.Containers = []v13.Container{
					{
						Resources: v13.ResourceRequirements{
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
						Resources: v13.ResourceRequirements{
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
				sku: &SKU{
					isUpdated: false,
				},
			},
			args: args{obj: getDeployment(BackendListenerName, func(d *appsv1.Deployment) {
				replica := int32(2)
				d.Spec.Replicas = &replica
				d.Spec.Template.Spec.Containers = []v13.Container{
					{
						Resources: v13.ResourceRequirements{
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
			args: args{obj: &v13.ConfigMap{}},
			fields: fields{
				sku: &SKU{
					isUpdated: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &AProductConfig{
				productName:     tt.fields.productName,
				resourceConfigs: tt.fields.resourceConfigs,
				sku:             tt.fields.sku,
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
		ObjectMeta: v12.ObjectMeta{
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
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		Spec: v1.DeploymentConfigSpec{
			Template: &v13.PodTemplateSpec{
				Spec: v13.PodSpec{
					Containers: []v13.Container{},
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
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: v13.PodTemplateSpec{
				Spec: v13.PodSpec{
					Containers: []v13.Container{},
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
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: v13.PodTemplateSpec{
				Spec: v13.PodSpec{
					Containers: []v13.Container{},
				},
			},
		}}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func getSKUConfig(modifyFn func(*v13.ConfigMap)) *v13.ConfigMap {
	mock := &v13.ConfigMap{
		ObjectMeta: v12.ObjectMeta{Name: ConfigMapName},
	}
	mock.Data = map[string]string{
		ConfigMapData: "[{\"name\": \"" + DEVSKU + "\",\"rate-limiting\": {\"unit\": \"minute\",\"requests_per_unit\": 1, \"alert_limits\": []},\"resources\": {\"" + ApicastProductionName + "\": {\"replicas\": 1,\"resources\": {\"requests\": {\"cpu\": \"50m\",\"memory\": \"50Mi\"},\"limits\": {\"cpu\": \"150m\",\"memory\": \"100Mi\"}}}}}, {\"name\": \"" + TWENTYMILLIONSKU + "\",\"rate-limiting\": {  \"unit\": \"minute\",  \"requests_per_unit\": 347,  \"alert_limits\": []},\"resources\": {\"" + BackendListenerName + "\": {\"replicas\": 3,\"resources\": {  \"requests\": {\"cpu\": 0.25,\"memory\": 450  },  \"limits\": {\"cpu\": 0.3,\"memory\": 500}}}}}]",
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}
