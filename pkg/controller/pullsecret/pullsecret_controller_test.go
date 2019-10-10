package pullsecret

import (
	"bytes"
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNameSpace = "test-namespace"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func basicReconcileRequest() reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: testNameSpace,
			Name:      testNameSpace,
		},
	}
}

func TestPullSecretReconciler(t *testing.T) {

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}

	defPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DefaultOriginPullSecretName,
			Namespace: resources.DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	scenarios := []struct {
		Name         string
		Request      reconcile.Request
		APINameSpace *corev1.Namespace
		FakeClient   client.Client
		Verify       func(client client.Client, res reconcile.Result, err error, t *testing.T)
	}{
		{
			Name:    "Pull Secret Controller does NOT add pull secret to namespace without WebApp label",
			Request: basicReconcileRequest(),
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNameSpace,
					Name:      testNameSpace,
				},
			}, defPullSecret),
			Verify: func(c client.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				// Secret should not be created - therefore should return an error when trying to find secret in the namespace
				s := &corev1.Secret{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: resources.DefaultOriginPullSecretName, Namespace: testNameSpace}, s)

				if err == nil {
					t.Fatal("expected err but got none")
				}
			},
		},
		{
			Name:    "Pull Secret Controller does add pull secret to namespace with WebApp label",
			Request: basicReconcileRequest(),
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNameSpace,
					Name:      testNameSpace,
					Labels:    map[string]string{WebAppLabel: "true"},
				},
			}, defPullSecret),
			Verify: func(c client.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				// Secret should be created - therefore should not return an error when trying to find secret in the namespace
				s := &corev1.Secret{}
				err = c.Get(context.TODO(), client.ObjectKey{Name: resources.DefaultOriginPullSecretName, Namespace: testNameSpace}, s)
				if err != nil {
					t.Fatal("expected no error but got one", err)
				}

				if bytes.Compare(s.Data["test"], defPullSecret.Data["test"]) != 0 {
					t.Fatalf("expected data %v, but got %v", defPullSecret.Data["test"], s.Data["test"])
				}
			},
		},
		{
			Name:    "Test get default pull secert error",
			Request: basicReconcileRequest(),
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNameSpace,
					Name:      testNameSpace,
					Labels:    map[string]string{WebAppLabel: "true"},
				},
			}),
			Verify: func(c client.Client, res reconcile.Result, err error, t *testing.T) {
				if err == nil {
					t.Fatalf("Expexted secret not found error but was nil")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {

			reconciler := ReconcilePullSecret{
				client: scenario.FakeClient,
				scheme: scheme,
			}

			res, err := reconciler.Reconcile(scenario.Request)
			scenario.Verify(scenario.FakeClient, res, err, t)
		})
	}
}
