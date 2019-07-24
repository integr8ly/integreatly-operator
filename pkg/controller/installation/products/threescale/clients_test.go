package threescale

import (
	"context"
	"net/http"

	"github.com/RHsyseng/operator-utils/pkg/olm"
	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/client"
	appsv1 "github.com/openshift/api/apps/v1"
	fakeappsv1Client "github.com/openshift/client-go/apps/clientset/versioned/fake"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeappsv1TypedClient "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1/fake"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
)

func getSigClient(preReqObjects []runtime.Object, scheme *runtime.Scheme) *client.SigsClientInterfaceMock {
	sigsFakeClient := client.NewSigsClientMoqWithScheme(scheme, preReqObjects...)
	sigsFakeClient.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
		switch obj := obj.(type) {
		case *corev1.Namespace:
			obj.Status.Phase = corev1.NamespaceActive
			return sigsFakeClient.GetSigsClient().Create(ctx, obj)
		case *threescalev1.APIManager:
			obj.Status.Deployments = olm.DeploymentStatus{
				Ready:    []string{"Ready status is when there is at least one ready and none starting or stopped"},
				Starting: []string{},
				Stopped:  []string{},
			}
			return sigsFakeClient.GetSigsClient().Create(ctx, obj)
		case *coreosv1alpha1.Subscription:
			obj.Status = coreosv1alpha1.SubscriptionStatus{
				Install: &coreosv1alpha1.InstallPlanReference{
					Name: installPlanFor3ScaleSubscription.Name,
				},
			}
			err := sigsFakeClient.GetSigsClient().Create(ctx, obj)
			if err != nil {
				return err
			}
			installPlanFor3ScaleSubscription.Namespace = obj.Namespace
			return sigsFakeClient.GetSigsClient().Create(ctx, installPlanFor3ScaleSubscription)
		}

		return sigsFakeClient.GetSigsClient().Create(ctx, obj)
	}

	return sigsFakeClient
}

func getAppsV1Client(appsv1PreReqs map[string]*appsv1.DeploymentConfig) appsv1Client.AppsV1Interface {
	fakeAppsv1 := fakeappsv1Client.NewSimpleClientset([]runtime.Object{}...).AppsV1()

	// Remove the generic reactor
	fakeAppsv1.(*fakeappsv1TypedClient.FakeAppsV1).Fake.ReactionChain = []testing.Reactor{}

	// The default NewSimpleClientset implementation does not handle 'instantiate' invocations correctly.
	// This implementation updates the status.latestVersion to record the action.
	fakeAppsv1.(*fakeappsv1TypedClient.FakeAppsV1).Fake.PrependReactor("create", "deploymentconfigs", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		switch action := action.(type) {
		case testing.CreateActionImpl:
			if action.Subresource == "instantiate" {
				dc := appsv1PreReqs[action.Name]
				dc.Status.LatestVersion = dc.Status.LatestVersion + 1
			}
		}

		return true, nil, nil
	})

	// Add our own simple get
	fakeAppsv1.(*fakeappsv1TypedClient.FakeAppsV1).Fake.PrependReactor("get", "deploymentconfigs", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		switch action := action.(type) {
		case testing.GetActionImpl:
			return true, appsv1PreReqs[action.Name], nil
		}

		return true, nil, nil
	})
	return fakeAppsv1
}

func getThreeScaleClient() *ThreeScaleInterfaceMock {
	return &ThreeScaleInterfaceMock{
		AddSSOIntegrationFunc: func(data map[string]string, accessToken string) (response *http.Response, e error) {
			return &http.Response{
				StatusCode: http.StatusCreated,
			}, nil
		},
		GetAdminUserFunc: func(accessToken string) (user *User, e error) {
			return threeScaleAdminUser, nil
		},
		UpdateAdminPortalUserDetailsFunc: func(username string, email string, accessToken string) (response *http.Response, e error) {
			threeScaleAdminUser.UserDetails.Username = username
			threeScaleAdminUser.UserDetails.Email = email
			return &http.Response{
				StatusCode: http.StatusOK,
			}, nil
		},
	}
}
