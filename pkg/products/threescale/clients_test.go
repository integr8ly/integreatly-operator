package threescale

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/RHsyseng/operator-utils/pkg/olm"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/client"

	appsv1 "github.com/openshift/api/apps/v1"
	fakeappsv1Client "github.com/openshift/client-go/apps/clientset/versioned/fake"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeappsv1TypedClient "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1/fake"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getSigClient(preReqObjects []runtime.Object, scheme *runtime.Scheme) *client.SigsClientInterfaceMock {
	sigsFakeClient := client.NewSigsClientMoqWithScheme(scheme, preReqObjects...)
	sigsFakeClient.CreateFunc = func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
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
				InstallPlanRef: &corev1.ObjectReference{
					Name:      installPlanFor3ScaleSubscription.Name,
					Namespace: obj.Namespace,
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
	testUsers := &Users{
		Users: []*User{},
	}
	testAuthProviders := &AuthProviders{
		AuthProviders: []*AuthProvider{},
	}
	return &ThreeScaleInterfaceMock{
		AddAuthenticationProviderFunc: func(data map[string]string, accessToken string) (response *http.Response, e error) {
			testAuthProviders.AuthProviders = append(testAuthProviders.AuthProviders, &AuthProvider{
				ProviderDetails: AuthProviderDetails{
					Kind:                           data["kind"],
					Name:                           data["name"],
					ClientId:                       data["client_id"],
					ClientSecret:                   data["client_secret"],
					Site:                           data["site"],
					SkipSSLCertificateVerification: data["skip_ssl_certificate_verification"] == "true",
					Published:                      data["published"] == "true",
				},
			})
			return &http.Response{
				StatusCode: http.StatusCreated,
			}, nil
		},
		GetAuthenticationProvidersFunc: func(accessToken string) (providers *AuthProviders, e error) {
			return testAuthProviders, nil
		},
		GetAuthenticationProviderByNameFunc: func(name string, accessToken string) (provider *AuthProvider, e error) {
			for _, ap := range testAuthProviders.AuthProviders {
				if ap.ProviderDetails.Name == name {
					return ap, nil
				}
			}

			return nil, &tsError{message: "Authprovider not found", StatusCode: http.StatusNotFound}
		},
		GetUsersFunc: func(accessToken string) (users *Users, e error) {
			return testUsers, nil
		},
		GetUserFunc: func(userName string, accessToken string) (user *User, e error) {
			for _, user := range testUsers.Users {
				if user.UserDetails.Username == userName {
					return user, nil
				}
			}

			return nil, fmt.Errorf("user %s not found", userName)
		},
		AddUserFunc: func(username string, email string, password string, accessToken string) (response *http.Response, e error) {
			testUsers.Users = append(testUsers.Users, &User{
				UserDetails: UserDetails{
					Role:     memberRole,
					Id:       rand.Int(),
					Username: username,
					Email:    email,
				},
			})
			return &http.Response{
				StatusCode: http.StatusCreated,
			}, nil
		},
		SetUserAsAdminFunc: func(userId int, accessToken string) (response *http.Response, e error) {
			for _, user := range testUsers.Users {
				if user.UserDetails.Id == userId {
					user.UserDetails.Role = adminRole
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
			}, nil
		},
		SetUserAsMemberFunc: func(userId int, accessToken string) (response *http.Response, e error) {
			for _, user := range testUsers.Users {
				if user.UserDetails.Id == userId {
					user.UserDetails.Role = memberRole
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
			}, nil
		},
	}
}
