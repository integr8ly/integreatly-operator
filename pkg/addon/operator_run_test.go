package addon

import (
	"context"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func existingSubscription(name string) *string {
	return &name
}

func noSubscription() *string {
	return nil
}
