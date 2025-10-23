package addon

import (
	"context"
	"errors"
	"testing"

	clientMock "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var genericError = errors.New("generic error")

func TestGetSubscription(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name                    string
		InstallType             integreatlyv1alpha1.InstallationType
		SubscriptionName        *string
		ExpectSubscriptionFound bool
	}{
		{
			Name:                    "RHOAM Add-on",
			InstallType:             integreatlyv1alpha1.InstallationTypeManagedApi,
			SubscriptionName:        existingSubscription("addon-managed-api-service"),
			ExpectSubscriptionFound: true,
		},
		{
			Name:                    "OLM Installation / RHOAM",
			InstallType:             integreatlyv1alpha1.InstallationTypeManagedApi,
			SubscriptionName:        existingSubscription("integreatly"),
			ExpectSubscriptionFound: true,
		},
		{
			Name:                    "Local run / RHOAM",
			InstallType:             integreatlyv1alpha1.InstallationTypeManagedApi,
			SubscriptionName:        noSubscription(),
			ExpectSubscriptionFound: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			installation := &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:      "installation",
					Namespace: "rhoam-test-operator",
				},
			}

			initObjs := []runtime.Object{
				installation,
			}
			if scenario.SubscriptionName != nil {
				initObjs = append(initObjs, &operatorsv1alpha1.Subscription{
					ObjectMeta: v1.ObjectMeta{
						Name:      *scenario.SubscriptionName,
						Namespace: "rhoam-test-operator",
					},
				})
			}

			client := utils.NewTestClient(scheme, initObjs...)

			subscription, err := GetSubscription(context.TODO(), client, installation)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (subscription != nil) != scenario.ExpectSubscriptionFound {
				t.Errorf("unexpeted subscription presence. Expected %t, but got %t", scenario.ExpectSubscriptionFound, subscription != nil)
			}
		})
	}
}

func getSub(name string, labelName string, labelValue string) operatorsv1alpha1.Subscription {
	return operatorsv1alpha1.Subscription{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "redhat-rhoam-operator",
			Labels: map[string]string{
				labelName: labelValue,
			},
		},
	}
}

func TestCPaaSSubscription(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	validSub := &operatorsv1alpha1.SubscriptionList{
		Items: []operatorsv1alpha1.Subscription{
			getSub("sub1", "operators.coreos.com/managed-api-service.redhat-rhoam-operator", ""),
		},
	}
	nosubs := &operatorsv1alpha1.SubscriptionList{
		Items: []operatorsv1alpha1.Subscription{},
	}
	wrongLabel := &operatorsv1alpha1.SubscriptionList{
		Items: []operatorsv1alpha1.Subscription{
			getSub("sub1", "key", "value"),
		},
	}

	scenarios := []struct {
		Name          string
		ExpectedFound bool
		ExpectedError bool
		Client        k8sclient.Client
	}{
		{
			Name:          "test CPaaS subscription exists",
			ExpectedFound: true,
			ExpectedError: false,
			Client:        utils.NewTestClient(scheme, []runtime.Object{validSub}...),
		},
		{
			Name:          "test no subs exist",
			ExpectedFound: false,
			ExpectedError: false,
			Client:        utils.NewTestClient(scheme, []runtime.Object{nosubs}...),
		},
		{
			Name:          "test wrong label",
			ExpectedFound: false,
			ExpectedError: false,
			Client:        utils.NewTestClient(scheme, []runtime.Object{wrongLabel}...),
		},
		{
			Name:          "error getting list of subscriptions",
			ExpectedFound: false,
			ExpectedError: true,
			Client: &clientMock.SigsClientInterfaceMock{
				ListFunc: func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
					return genericError
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {

			sub, err := GetRhoamCPaaSSubscription(context.TODO(), scenario.Client, "redhat-rhoam-operator")
			if err != nil && !scenario.ExpectedError {
				t.Fatal("Got unexpected error", err)
			}
			if scenario.ExpectedFound && sub == nil {
				t.Fatal("Expected to find subscription")
			}
		})
	}
}

func existingSubscription(name string) *string {
	return &name
}

func noSubscription() *string {
	return nil
}

func TestOperatorHiveManaged(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name          string
		ExpectedError bool
		HiveManaged   bool
		Client        k8sclient.Client
		Installation  *integreatlyv1alpha1.RHMI
	}{
		{
			Name:          "test hive managed operator",
			ExpectedError: false,
			HiveManaged:   true,
			Client: utils.NewTestClient(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
						Labels: map[string]string{
							RhoamAddonInstallManagedLabel: "true",
						},
					},
				},
			),
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "redhat-rhoam-operator",
				},
			},
		},
		{
			Name:          "test not hive managed operator with managed false",
			ExpectedError: false,
			HiveManaged:   false,
			Client: utils.NewTestClient(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
						Labels: map[string]string{
							RhoamAddonInstallManagedLabel: "false",
						},
					},
				},
			),
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "redhat-rhoam-operator",
				},
			},
		},
		{
			Name:          "test not hive managed operator without label",
			ExpectedError: false,
			HiveManaged:   false,
			Client: utils.NewTestClient(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
					},
				},
			),
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "redhat-rhoam-operator",
				},
			},
		},
		{
			Name:          "test hive managed error if empty installation cr is found",
			ExpectedError: true,
			HiveManaged:   true,
			Client: utils.NewTestClient(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
						Labels: map[string]string{
							RhoamAddonInstallManagedLabel: "true",
						},
					},
				},
			),
			Installation: &integreatlyv1alpha1.RHMI{},
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.Name, func(t *testing.T) {
			isHiveManaged, err := OperatorIsHiveManaged(context.TODO(), tt.Client, tt.Installation)
			if err == nil && tt.ExpectedError {
				t.Fatal("expected error but found none")
			}
			if err != nil && !tt.ExpectedError {
				t.Fatal("error found but none expected")
			}
			if err == nil && isHiveManaged && !tt.HiveManaged {
				t.Fatal("error operator is not hive managed but reporting that it is")
			}
			if err == nil && !isHiveManaged && tt.HiveManaged {
				t.Fatal("error operator is hive managed but reporting that it's not")
			}
		})
	}
}

func TestInferOperatorRunType(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		client       k8sclient.Client
		installation *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    OperatorRunType
		wantErr bool
	}{
		{
			name: "failed to get subscription",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return genericError
					},
				},
				installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "infer operator run type from subscription",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &operatorsv1alpha1.Subscription{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name:      ManagedAPIService,
						Namespace: "ns",
					},
				}),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: false,
			want:    OLMRunType,
		},
		{
			name: "operator run type is cluster",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{
						Name:      "rhoam-operator",
						Namespace: "ns",
					},
				}),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			wantErr: false,
			want:    ClusterRunType,
		},
		{
			name: "fallback to local run type",
			args: args{
				ctx:    context.TODO(),
				client: utils.NewTestClient(scheme),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: false,
			want:    LocalRunType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferOperatorRunType(tt.args.ctx, tt.args.client, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferOperatorRunType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InferOperatorRunType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperatorInstalledViaOLM(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		client       k8sclient.Client
		installation *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "failed to infer operator run type",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return genericError
					},
				},
				installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "operator run type is olm",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &operatorsv1alpha1.Subscription{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name:      ManagedAPIService,
						Namespace: "ns",
					},
				}),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: false,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OperatorInstalledViaOLM(tt.args.ctx, tt.args.client, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Errorf("OperatorInstalledViaOLM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OperatorInstalledViaOLM() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCatalogSource(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		client       k8sclient.Client
		installation *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "failed to get subscription",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return genericError
					},
				},
				installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "rhoam cpaas subscription",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &operatorsv1alpha1.SubscriptionList{
					Items: []operatorsv1alpha1.Subscription{
						{
							ObjectMeta: v1.ObjectMeta{
								Namespace: "ns",
								Labels: map[string]string{
									"operators.coreos.com/managed-api-service.redhat-rhoam-operator": "test",
								},
							},
							Spec: &operatorsv1alpha1.SubscriptionSpec{},
						},
					},
				},
					&operatorsv1alpha1.CatalogSource{},
				),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "subscription is nil",
			args: args{
				ctx:    context.TODO(),
				client: utils.NewTestClient(scheme),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "retrieved catalog source from the subscription",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&operatorsv1alpha1.Subscription{
						ObjectMeta: v1.ObjectMeta{
							Name:      ManagedAPIService,
							Namespace: "ns",
						},
						Spec: &operatorsv1alpha1.SubscriptionSpec{},
					},
					&operatorsv1alpha1.CatalogSource{},
				),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: false,
			want:    "",
		},
		{
			name: "failed to retrieve catalog source",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme,
					&operatorsv1alpha1.Subscription{
						ObjectMeta: v1.ObjectMeta{
							Name:      ManagedAPIService,
							Namespace: "ns",
						},
						Spec: &operatorsv1alpha1.SubscriptionSpec{},
					},
				),
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: ManagedAPIService,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCatalogSource(tt.args.ctx, tt.args.client, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCatalogSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && got.Name != tt.want {
				t.Errorf("GetCatalogSource() got = %v, want %v", got, tt.want)
			}
		})
	}
}
