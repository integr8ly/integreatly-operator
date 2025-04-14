package controllers

import (
	"os"
	"reflect"
	"testing"
	"time"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	FakeName      = "fake-name"
	FakeNamespace = "fake-namespace"
	FakeHost      = "fake-route.org"
)

func TestHandleCROConfigDeletion(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		installation rhmiv1alpha1.RHMI
		wantErr      bool
	}{
		{
			name: "handle CRO config map deletion when config map exists",
			installation: rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		r := &RHMIReconciler{
			Client: utils.NewTestClient(scheme, getCROConfigMap()),
		}
		err := r.handleCROConfigDeletion(tt.installation)
		if (err != nil) != tt.wantErr {
			t.Errorf("handleCROConfigDeletion() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
	}
}

func TestFirstInstallFirstReconcile(t *testing.T) {
	tests := []struct {
		name         string
		installation *rhmiv1alpha1.RHMI
		want         bool
	}{
		{
			name: "test CR for first install, first reconcile",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "",
				},
			},
			want: true,
		},
		{
			name: "test CR for first install, installation complete",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "",
				},
			},
			want: false,
		},
		{
			name: "test CR for first install, installation in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
		{
			name: "test CR for installation complete, upgrade in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		got := firstInstallFirstReconcile(tt.installation)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("firstInstallFirstReconcile() got = %v, want %v", got, tt.want)
		}
	}
}

func TestUpgradeFirstReconcile(t *testing.T) {
	tests := []struct {
		name         string
		installation *rhmiv1alpha1.RHMI
		want         bool
	}{
		{
			name: "test CR for first install, first reconcile",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "",
				},
			},
			want: false,
		},
		{
			name: "test CR for first install, installation complete",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "",
				},
			},
			want: true,
		},
		{
			name: "test CR for first install, installation in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
		{
			name: "test CR for installation complete, upgrade in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		got := upgradeFirstReconcile(tt.installation)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("upgradeFirstReconcile() got = %v, want %v", got, tt.want)
		}
	}
}

func getCROConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultCloudResourceConfigName,
			Namespace: FakeNamespace,
			Finalizers: []string{
				deletionFinalizer,
			},
		},
	}
}

func Test_getInstallation(t *testing.T) {
	tests := []struct {
		name    string
		want    *rhmiv1alpha1.RHMI
		wantErr bool
		envs    map[string]string
	}{
		{
			name:    "WATCH_NAMESPACE must be set",
			wantErr: true,
			want:    nil,
		},
		{
			name:    "INSTALLATION_TYPE not set",
			wantErr: false,
			want: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace",
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: "",
				},
			},
			envs: map[string]string{"WATCH_NAMESPACE": "namespace"},
		},
		{
			name:    "INSTALLATION_TYPE is set",
			wantErr: false,
			want: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace",
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: "managed-api-service",
				},
			},
			envs: map[string]string{
				"WATCH_NAMESPACE":   "namespace",
				"INSTALLATION_TYPE": "managed-api-service",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preEnv := make(map[string]string)
			envs := []string{
				"WATCH_NAMESPACE",
				"INSTALLATION_TYPE",
			}

			for _, env := range envs {
				_, ok := tt.envs[env]
				if !ok {

					preEnv[env] = os.Getenv(env)
					err := os.Unsetenv(env)
					if err != nil {
						t.Error("error unsetting env var : ", err)
					}
				}
			}

			t.Cleanup(func() {
				for key, value := range preEnv {
					err := os.Setenv(key, value)
					if err != nil {
						t.Error("error setting env var : ", err)
					}
				}
			})

			for key, value := range tt.envs {
				t.Setenv(key, value)
			}

			got, err := getInstallation()
			if (err != nil) != tt.wantErr {
				t.Errorf("getInstallation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getInstallation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isInstallationOlderThan1Minute(t *testing.T) {
	type args struct {
		installation *rhmiv1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Installation is older than a minute",
			want: true,
			args: args{installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{
						Time: time.Now().Add(-2 * time.Minute),
					},
				},
			}},
		},
		{
			name: "Installation is newer than a minute",
			want: false,
			args: args{installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{
						Time: time.Now().Add(-59 * time.Second),
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInstallationOlderThan1Minute(tt.args.installation); got != tt.want {
				t.Errorf("isInstallationOlderThan1Minute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRebalancePods(t *testing.T) {
	tests := []struct {
		name string
		want bool
		envs map[string]string
	}{
		{
			name: "REBALANCE_PODS does not exist",
			want: true,
		},
		{
			name: "REBALANCE_PODS is set to false",
			want: false,
			envs: map[string]string{
				"REBALANCE_PODS": "false",
			},
		},
		{
			name: "REBALANCE_PODS is set to true",
			want: true,
			envs: map[string]string{
				"REBALANCE_PODS": "true",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.envs {
				t.Setenv(key, value)
			}
			if got := getRebalancePods(); got != tt.want {
				t.Errorf("getRebalancePods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRHMIReconciler_checkClusterPackageAvailability(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Client                     client.Client
		Scheme                     *runtime.Scheme
		mgr                        controllerruntime.Manager
		controller                 controller.Controller
		restConfig                 *rest.Config
		customInformers            map[string]map[string]*cache.Informer
		productsInstallationLoader marketplace.ProductsInstallationLoader
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test - RHOAM - cluster package successfully reconciled",
			fields: fields{
				Client: utils.NewTestClient(scheme,
					&packageOperatorv1alpha1.ClusterPackage{
						ObjectMeta: metav1.ObjectMeta{
							Name: "managed-api-service",
							Generation: 1,
						},
						Status: packageOperatorv1alpha1.PackageStatus{
							Conditions: []metav1.Condition{
								metav1.Condition{
									Type: "Available",
									Status: metav1.ConditionTrue,
									ObservedGeneration: 1,
								},
								metav1.Condition{
									Type: "Progressing",
									Status: metav1.ConditionFalse,
									ObservedGeneration: 1,
								},
							},
						},
					},
				)},
			wantErr: false,
		},
		{
			name: "Test - RHOAM - cluster package successfully reconciled but with wrong generation being available",
			fields: fields{
				Client: utils.NewTestClient(scheme,
					&packageOperatorv1alpha1.ClusterPackage{
						ObjectMeta: metav1.ObjectMeta{
							Name: "managed-api-service",
							Generation: 1,
						},
						Status: packageOperatorv1alpha1.PackageStatus{
							Conditions: []metav1.Condition{
								metav1.Condition{
									Type: "Available",
									Status: metav1.ConditionTrue,
									ObservedGeneration: 0,
								},
								metav1.Condition{
									Type: "Progressing",
									Status: metav1.ConditionFalse,
									ObservedGeneration: 0,
								},
							},
						},
					},
				)},
			wantErr: true,
		},
		{
			name: "Test - RHOAM - cluster package is not available",
			fields: fields{
				Client: utils.NewTestClient(scheme,
					&packageOperatorv1alpha1.ClusterPackage{
						ObjectMeta: metav1.ObjectMeta{
							Name: "managed-api-service",
							Generation: 1,
						},
						Status: packageOperatorv1alpha1.PackageStatus{
							Conditions: []metav1.Condition{
								metav1.Condition{
									Type: "Available",
									Status: metav1.ConditionFalse,
									ObservedGeneration: 0,
								},
								metav1.Condition{
									Type: "Progressing",
									Status: metav1.ConditionTrue,
									ObservedGeneration: 1,
								},
							},
						},
					},
				)},
			wantErr: true,
		},
		{
			name: "Test - RHOAM - incorrect cluster package name",
			fields: fields{
				Client: utils.NewTestClient(scheme,
					&packageOperatorv1alpha1.ClusterPackage{
						ObjectMeta: metav1.ObjectMeta{
							Name: "incorrect-name",
						},
						Status: packageOperatorv1alpha1.PackageStatus{
							Conditions: []metav1.Condition{
								metav1.Condition{
									Type: "Available",
									Status: metav1.ConditionTrue,
								},
								metav1.Condition{
									Type: "Progressing",
									Status: metav1.ConditionFalse,
								},
							},
						},
					},
				)},
			wantErr: true,
		},
		{
			name: "Test - RHOAM - cluster package not found",
			fields: fields{
				Client: utils.NewTestClient(scheme)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Setenv("WATCH_NAMESPACE", "test")
		t.Run(tt.name, func(t *testing.T) {

			r := &RHMIReconciler{
				Client:                     tt.fields.Client,
				Scheme:                     tt.fields.Scheme,
				mgr:                        tt.fields.mgr,
				controller:                 tt.fields.controller,
				restConfig:                 tt.fields.restConfig,
				customInformers:            tt.fields.customInformers,
				productsInstallationLoader: tt.fields.productsInstallationLoader,
			}

			err := r.checkClusterPackageAvailablity()
			if (err != nil) != tt.wantErr {
				t.Errorf("getAlertingNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
