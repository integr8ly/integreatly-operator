package dynamic_resources

import (
	kcTypes "github.com/integr8ly/keycloak-client/pkg/types"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestKCRealmFunctionality(t *testing.T) {
	scheme := runtime.NewScheme()

	kcRealm := kcTypes.KeycloakRealm{
		TypeMeta: metav1.TypeMeta{
			Kind:       kcTypes.KeycloakRealmKind,
			APIVersion: kcTypes.KeycloakRealmApiVersion,
		},
	}

	kcRealmUnstructed, err := ConvertKeycloakRealmTypedToUnstructured(&kcRealm)
	if err != nil {
		t.FailNow()
	}

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   kcTypes.KeycloakRealmGroup,
		Version: kcTypes.KeycloakRealmVersion,
		Kind:    kcTypes.KeycloakRealmKind,
	},
		kcRealmUnstructed)

	kcRealmOriginal := kcTypes.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example-ns",
		},
		Spec: kcTypes.KeycloakRealmSpec{
			Unmanaged: true,
		},
	}

	kcRealmOriginalUnstructured, err := ConvertKeycloakRealmTypedToUnstructured(&kcRealmOriginal)
	if err != nil {
		t.FailNow()
	}

	kcRealmDesired := kcTypes.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example-ns",
		},
		Spec: kcTypes.KeycloakRealmSpec{
			Unmanaged: false,
		},
	}

	tests := []struct {
		name         string
		serverClient k8sclient.Client
	}{
		{
			name:         "Test KC realm functions",
			serverClient: fakeclient.NewFakeClientWithScheme(scheme, kcRealmOriginalUnstructured),
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			op, kc, err := CreateOrUpdateKeycloakRealm(context.TODO(), tt.serverClient, kcRealmDesired)
			if err != nil {
				t.FailNow()
			}
			if op != controllerutil.OperationResultUpdated {
				t.FailNow()
			}
			if kc.Spec.Unmanaged != false {
				t.FailNow()
			}
		})
	}
}

func TestKCFunctionality(t *testing.T) {
	scheme := runtime.NewScheme()

	kc := kcTypes.Keycloak{
		TypeMeta: metav1.TypeMeta{
			Kind:       kcTypes.KeycloakKind,
			APIVersion: kcTypes.KeycloakApiVersion,
		},
	}

	kcUnstructed, err := ConvertKeycloakTypedToUnstructured(&kc)
	if err != nil {
		t.FailNow()
	}

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   kcTypes.KeycloakGroup,
		Version: kcTypes.KeycloakVersion,
		Kind:    kcTypes.KeycloakKind,
	},
		kcUnstructed)

	kcOriginal := kcTypes.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example-ns",
		},
		Spec: kcTypes.KeycloakSpec{
			Unmanaged: true,
		},
	}

	kcOriginalUnstructured, err := ConvertKeycloakTypedToUnstructured(&kcOriginal)
	if err != nil {
		t.FailNow()
	}

	kcDesired := kcTypes.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example-ns",
		},
		Spec: kcTypes.KeycloakSpec{
			Unmanaged: false,
		},
	}

	tests := []struct {
		name         string
		serverClient k8sclient.Client
	}{
		{
			name:         "Test KC functions",
			serverClient: fakeclient.NewFakeClientWithScheme(scheme, kcOriginalUnstructured),
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			op, kc, err := CreateOrUpdateKeycloak(context.TODO(), tt.serverClient, kcDesired)
			if err != nil {
				t.FailNow()
			}
			if op != controllerutil.OperationResultUpdated {
				t.FailNow()
			}
			if kc.Spec.Unmanaged != false {
				t.FailNow()
			}
		})
	}
}

func TestKCUserFunctionality(t *testing.T) {
	scheme := runtime.NewScheme()

	kcUser := kcTypes.KeycloakUser{
		TypeMeta: metav1.TypeMeta{
			Kind:       kcTypes.KeycloakUserKind,
			APIVersion: kcTypes.KeycloakUserApiVersion,
		},
	}

	kcUserUnstructed, err := ConvertKeycloakUserTypedToUnstructured(&kcUser)
	if err != nil {
		t.FailNow()
	}

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   kcTypes.KeycloakUserGroup,
		Version: kcTypes.KeycloakUserVersion,
		Kind:    kcTypes.KeycloakUserKind,
	},
		kcUserUnstructed)

	kcUserOriginal := kcTypes.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example-ns",
		},
		Spec: kcTypes.KeycloakUserSpec{
			User: kcTypes.KeycloakAPIUser{
				UserName: "testUser",
			},
		},
	}

	kcUserOriginalUnstructured, err := ConvertKeycloakUserTypedToUnstructured(&kcUserOriginal)
	if err != nil {
		t.FailNow()
	}

	kcUserDesired := kcTypes.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example-ns",
		},
		Spec: kcTypes.KeycloakUserSpec{
			User: kcTypes.KeycloakAPIUser{
				UserName: "testUserDesired",
			},
		},
	}

	tests := []struct {
		name         string
		serverClient k8sclient.Client
	}{
		{
			name:         "Test KC User functions",
			serverClient: fakeclient.NewFakeClientWithScheme(scheme, kcUserOriginalUnstructured),
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			op, kc, err := CreateOrUpdateKeycloakUser(context.TODO(), tt.serverClient, kcUserDesired)
			if err != nil {
				t.FailNow()
			}
			if op != controllerutil.OperationResultUpdated {
				t.FailNow()
			}
			if kc.Spec.User.UserName != "testUserDesired" {
				t.FailNow()
			}
		})
	}
}