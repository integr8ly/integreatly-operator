package fuse

import (
	"context"
	"fmt"
	"testing"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/pkg/errors"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var ip = &coreosv1alpha1.InstallPlan{
	ObjectMeta: metav1.ObjectMeta{
		Name: "fuse-installplan",
	},
	Status: coreosv1alpha1.InstallPlanStatus{
		Phase: coreosv1alpha1.InstallPlanPhaseComplete,
	},
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadFuseFunc: func() (ready *config.Fuse, e error) {
			return config.NewFuse(config.ProductConfig{}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "fuse",
				"URL":       "fuse.openshift-cluster.com",
			}), nil
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = aerogearv1.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = syn.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestFuse(t *testing.T) {
	// set up the fake client
	ctx := context.TODO()
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error getting scheme : %v", err)
	}
	fakeClient := getSigClient([]runtime.Object{}, scheme)

	// create Installation and start reconciler
	installation := &v1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-installation",
			Namespace: "integreatly-operator-namespace",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Spec: v1alpha1.InstallationSpec{
			MasterURL:        "https://console.apps.example.com",
			RoutingSubdomain: "apps.example.com",
		},
	}

	testReconciler, err := NewReconciler(basicConfigMock(), installation, marketplace.NewManager())
	status, err := testReconciler.Reconcile(ctx, installation, fakeClient)
	if err != nil {
		t.Fatalf("Error reconciling fuse: %v", err)
	}

	if status != v1alpha1.PhaseCompleted {
		t.Fatalf("unexpected status: %v, expected: %v", status, v1alpha1.PhaseCompleted)
	}

	err = assertReconciled(installation, fakeClient)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func assertReconciled(inst *v1alpha1.Installation, fakeSigsClient *client.SigsClientInterfaceMock) error {
	ctx := context.TODO()

	// expect the namespace to be created
	ns := &corev1.Namespace{}
	err := fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: defaultInstallationNamespace}, ns)
	if k8serr.IsNotFound(err) {
		return errors.New(fmt.Sprintf("%s namespace was not created", defaultInstallationNamespace))
	}

	// expect the subscription to be created
	sub := &coreosv1alpha1.Subscription{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: defaultSubscriptionName, Namespace: defaultInstallationNamespace}, sub)
	if k8serr.IsNotFound(err) {
		return errors.New("Fuse operator subscription was not created")
	}

	// expect the custom resource to be created
	cr := &syn.Syndesis{}
	err = fakeSigsClient.Get(ctx, pkgclient.ObjectKey{Name: "integreatly", Namespace: defaultInstallationNamespace}, cr)
	if k8serr.IsNotFound(err) {
		return errors.New("Fuse custom resource was not created")
	}

	return nil
}

func getSigClient(preReqObjects []runtime.Object, scheme *runtime.Scheme) *client.SigsClientInterfaceMock {
	sigsFakeClient := client.NewSigsClientMoqWithScheme(scheme, preReqObjects...)
	sigsFakeClient.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
		switch obj := obj.(type) {
		case *corev1.Namespace:
			obj.Status.Phase = corev1.NamespaceActive
			return sigsFakeClient.GetSigsClient().Create(ctx, obj)
		case *syn.Syndesis:
			obj.Status.Phase = syn.SyndesisPhaseInstalled
			return sigsFakeClient.GetSigsClient().Create(ctx, obj)

		case *coreosv1alpha1.Subscription:
			obj.Status = coreosv1alpha1.SubscriptionStatus{
				Install: &coreosv1alpha1.InstallPlanReference{
					Name: ip.Name,
				},
			}
			err := sigsFakeClient.GetSigsClient().Create(ctx, obj)
			if err != nil {
				return err
			}
			ip.Namespace = obj.Namespace
			return sigsFakeClient.GetSigsClient().Create(ctx, ip)
		}

		return sigsFakeClient.GetSigsClient().Create(ctx, obj)
	}

	return sigsFakeClient
}
