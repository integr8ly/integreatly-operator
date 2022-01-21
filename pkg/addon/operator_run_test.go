package addon

import (
	"context"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetSubscription(t *testing.T) {
	scheme := runtime.NewScheme()
	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	operatorsv1alpha1.AddToScheme(scheme)

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
			Name:                    "RHMI Add-on",
			InstallType:             integreatlyv1alpha1.InstallationTypeManaged,
			SubscriptionName:        existingSubscription("addon-rhmi"),
			ExpectSubscriptionFound: true,
		},
		{
			Name:                    "OLM Installation / RHOAM",
			InstallType:             integreatlyv1alpha1.InstallationTypeManagedApi,
			SubscriptionName:        existingSubscription("integreatly"),
			ExpectSubscriptionFound: true,
		},
		{
			Name:                    "OLM Installation / RHMI",
			InstallType:             integreatlyv1alpha1.InstallationTypeManaged,
			SubscriptionName:        existingSubscription("integreatly"),
			ExpectSubscriptionFound: true,
		},
		{
			Name:                    "Local run / RHOAM",
			InstallType:             integreatlyv1alpha1.InstallationTypeManagedApi,
			SubscriptionName:        noSubscription(),
			ExpectSubscriptionFound: false,
		},
		{
			Name:                    "Local run / RHMI",
			InstallType:             integreatlyv1alpha1.InstallationTypeManaged,
			SubscriptionName:        noSubscription(),
			ExpectSubscriptionFound: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			installation := &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:      "installation",
					Namespace: "rhmi-test-operator",
				},
			}

			initObjs := []runtime.Object{
				installation,
			}
			if scenario.SubscriptionName != nil {
				initObjs = append(initObjs, &operatorsv1alpha1.Subscription{
					ObjectMeta: v1.ObjectMeta{
						Name:      *scenario.SubscriptionName,
						Namespace: "rhmi-test-operator",
					},
				})
			}

			client := fake.NewFakeClientWithScheme(scheme, initObjs...)

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
	scheme := runtime.NewScheme()
	operatorsv1alpha1.AddToScheme(scheme)
	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)

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
		Client        client.Client
	}{
		{
			Name:          "test CPaaS subscription exists",
			ExpectedFound: true,
			ExpectedError: false,
			Client:        fake.NewFakeClientWithScheme(scheme, []runtime.Object{validSub}...),
		},
		{
			Name:          "test no subs exist",
			ExpectedFound: false,
			ExpectedError: false,
			Client:        fake.NewFakeClientWithScheme(scheme, []runtime.Object{nosubs}...),
		},
		{
			Name:          "test wrong label",
			ExpectedFound: false,
			ExpectedError: false,
			Client:        fake.NewFakeClientWithScheme(scheme, []runtime.Object{wrongLabel}...),
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
	scheme := runtime.NewScheme()
	corev1.SchemeBuilder.AddToScheme(scheme)
	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	scenarios := []struct {
		Name          string
		ExpectedError bool
		HiveManaged   bool
		Client        client.Client
		Installation  *integreatlyv1alpha1.RHMI
	}{
		{
			Name:          "test hive managed operator",
			ExpectedError: false,
			HiveManaged:   true,
			Client: fake.NewFakeClientWithScheme(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
						Labels: map[string]string{
							"hive.openshift.io/managed": "true",
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
			Client: fake.NewFakeClientWithScheme(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
						Labels: map[string]string{
							"hive.openshift.io/managed": "false",
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
			Client: fake.NewFakeClientWithScheme(scheme,
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
			Client: fake.NewFakeClientWithScheme(scheme,
				&corev1.Namespace{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: "redhat-rhoam-operator",
						Labels: map[string]string{
							"hive.openshift.io/managed": "true",
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
