package launcher

import (
	"context"

	launcherv1alpha2 "github.com/fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"
	"github.com/integr8ly/integreatly-operator/pkg/client"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func getSigClient(preReqObjects []runtime.Object, scheme *runtime.Scheme) *client.SigsClientInterfaceMock {
	sigsFakeClient := client.NewSigsClientMoqWithScheme(scheme, preReqObjects...)
	sigsFakeClient.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
		switch obj := obj.(type) {
		case *corev1.Namespace:
			obj.Status.Phase = corev1.NamespaceActive
			return sigsFakeClient.GetSigsClient().Create(ctx, obj)
		case *coreosv1alpha1.Subscription:
			obj.Status = coreosv1alpha1.SubscriptionStatus{
				Install: &coreosv1alpha1.InstallPlanReference{
					Name: installPlanForLauncherSubscription.Name,
				},
			}
			err := sigsFakeClient.GetSigsClient().Create(ctx, obj)
			if err != nil {
				return err
			}
			installPlanForLauncherSubscription.Namespace = obj.Namespace
			return sigsFakeClient.GetSigsClient().Create(ctx, installPlanForLauncherSubscription)
		case *launcherv1alpha2.Launcher:
			err := sigsFakeClient.GetSigsClient().Create(ctx, mockLauncherRoute)
			if err != nil {
				return err
			}
		}

		return sigsFakeClient.GetSigsClient().Create(ctx, obj)
	}

	return sigsFakeClient
}
