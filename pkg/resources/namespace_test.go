package resources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestReconcileNamespace(t *testing.T) {
	commonValidate := func(t *testing.T, ns *v1.Namespace) {
		if ns == nil {
			t.Fatal("expected a Namespace but got nil ")
		}
		if ns.Name != "something" {
			t.Fatal("expected the namespace name to be 'something' but got ", ns.Name)
		}
		if _, ok := ns.Labels["integreatly"]; !ok {
			t.Fatal("expected an integreatly label but it was not present")
		}
		if len(ns.OwnerReferences) == 0 {
			t.Fatal("expected there to be an owner ref set but there was none")
		}
	}
	cases := []struct {
		Name                 string
		NS                   *v1.Namespace
		Owner                *v1alpha1.Installation
		FakeControllerClient func() client.Client
		ExpectError          bool
		Validate             func(t *testing.T, ns *v1.Namespace)
	}{
		{
			Name: "test namespace reconciled correctly when not already created",
			NS:   &v1.Namespace{ObjectMeta: v12.ObjectMeta{Name: "something"}},
			Owner: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					Kind:       "Installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			FakeControllerClient: func() client.Client {
				c := pkgclient.NewFakeClient()
				return c
			},
			Validate: commonValidate,
		},
		{
			Name: "test namespace reconciled correctly when already created",
			NS: &v1.Namespace{ObjectMeta: v12.ObjectMeta{Name: "something", Labels: map[string]string{"integreatly": "true"}, OwnerReferences: []v12.OwnerReference{
				{
					Name:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "Installation",
				},
			}}},
			Owner: &v1alpha1.Installation{
				ObjectMeta: v12.ObjectMeta{
					Name: "installation",
				},
				TypeMeta: v12.TypeMeta{
					Kind:       "Installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			FakeControllerClient: func() client.Client {
				c := pkgclient.NewFakeClient(&v1.Namespace{
					ObjectMeta: v12.ObjectMeta{Name: "something"},
				})
				return c
			},
			Validate: commonValidate,
		},
		{
			Name:  "test existing namespace that is not ours causes error",
			NS:    &v1.Namespace{ObjectMeta: v12.ObjectMeta{Name: "something"}},
			Owner: &v1alpha1.Installation{},
			FakeControllerClient: func() client.Client {
				c := pkgclient.NewFakeClient(&v1.Namespace{
					ObjectMeta: v12.ObjectMeta{Name: "something"},
				})
				return c
			},
			ExpectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			nsReconcile := NewNamespaceReconciler(tc.FakeControllerClient())
			ns, err := nsReconcile.Reconcile(context.TODO(), tc.NS, tc.Owner)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("did not expect an error but got one ", err)
			}
			if tc.Validate != nil {
				tc.Validate(t, ns)
			}
		})
	}
}
